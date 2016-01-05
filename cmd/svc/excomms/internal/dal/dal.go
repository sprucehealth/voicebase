package dal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
)

type ProvisionedNumberLookup struct {
	PhoneNumber    *string
	ProvisionedFor *string
}

type OutgoingCallRouteLookup struct {
	CallSID *string
}

type DAL interface {

	// LookupProvisionedPhoneNumber returns a provisioned phone number based on the lookup query.
	LookupProvisionedPhoneNumber(lookup *ProvisionedNumberLookup) (*models.ProvisionedPhoneNumber, error)

	// ProvisionPhoneNumber provisions the provided phone number.
	ProvisionPhoneNumber(ppn *models.ProvisionedPhoneNumber) error

	// LogEvent persists the provided event for operational purposes.
	// TODO: If this data gets noisy, might make sense to log this data into a different
	// database of its own.
	LogEvent(e *models.Event) error

	// CreateCallRequest creates the provided call request.
	CreateCallRequest(*models.CallRequest) error

	// UpdateCallRequest updates an unexpired call request originating from the source phone number with the
	// provided call SID.
	UpdateCallRequest(sourcePhoneNumber, callSID string) (int64, error)

	// ValidCallRequest returns an unexpired call request from the sourcePhoneNumber.
	// TODO: This assumes that there can be only a single valid call request at a time for a given
	// source phone number. Update this logic to be more sophisticated and account for multiple
	// valid call requests being proxied via different phone numbers.
	ValidCallRequest(sourcePhoneNumber string) (*models.CallRequest, error)

	// LookupCallRequest returns the call request identified by the call sid.
	LookupCallRequest(callSID string) (*models.CallRequest, error)
}

type dal struct {
	db *sql.DB
}

func NewDAL(db *sql.DB) DAL {
	return &dal{
		db: db,
	}
}

var ErrProvisionedNumberNotFound = errors.New("provisioned number not found")

func (d *dal) LookupProvisionedPhoneNumber(lookup *ProvisionedNumberLookup) (*models.ProvisionedPhoneNumber, error) {
	var ppn models.ProvisionedPhoneNumber
	var where string
	var val interface{}

	if lookup.PhoneNumber != nil {
		where = "phone_number = ?"
		val = *lookup.PhoneNumber
	} else if lookup.ProvisionedFor != nil {
		where = "provisioned_for = ?"
		val = *lookup.ProvisionedFor
	} else {
		return nil, errors.Trace(fmt.Errorf("phone_number or provisioned_for required to lookup provisioned phone number"))
	}

	err := d.db.QueryRow(`
		SELECT phone_number, provisioned_for, created 
		FROM provisioned_phone_number 
		WHERE `+where, val).Scan(
		&ppn.PhoneNumber,
		&ppn.ProvisionedFor,
		&ppn.Provisioned)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrProvisionedNumberNotFound)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &ppn, nil
}

func (d *dal) ProvisionPhoneNumber(ppn *models.ProvisionedPhoneNumber) error {
	if ppn == nil {
		return nil
	}

	_, err := d.db.Exec(`
		INSERT INTO provisioned_phone_number (phone_number, provisioned_for) VALUES (?,?)`, ppn.PhoneNumber, ppn.ProvisionedFor)
	return errors.Trace(err)
}

func (d *dal) LogEvent(e *models.Event) error {
	if e == nil {
		return nil
	}

	data, err := json.Marshal(e)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = d.db.Exec(`
		INSERT INTO excomms_event (source, destination, data, event)
		VALUES (?,?,?,?)`, e.Source, e.Destination, data, e.Type)

	return errors.Trace(err)
}

func (d *dal) CreateCallRequest(cr *models.CallRequest) error {
	if cr == nil {
		return nil
	}

	_, err := d.db.Exec(`
		INSERT INTO outgoing_call_request (source, destination, proxy, organization_id, requested, expires)
		VALUES (?,?,?,?,?,?)`, cr.Source, cr.Destination, cr.Proxy, cr.OrganizationID, cr.Requested, cr.Expires)
	return errors.Trace(err)
}

var ErrCallRequestNotFound = errors.New("call request not found")

func (d *dal) UpdateCallRequest(sourcePhoneNumber, callSID string) (int64, error) {
	res, err := d.db.Exec(`
		UPDATE outgoing_call_request
		SET call_sid = ?
		WHERE source = ?
		AND expires > ?`, callSID, sourcePhoneNumber, time.Now())
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsAffected, err := res.RowsAffected()
	return rowsAffected, errors.Trace(err)
}

func (d *dal) ValidCallRequest(sourcePhoneNumber string) (*models.CallRequest, error) {
	var cr models.CallRequest

	err := d.db.QueryRow(`
		SELECT source, destination, proxy, organization_id, requested, expires, call_sid 
		FROM outgoing_call_request
		WHERE source = ?
		AND expires > ?`, sourcePhoneNumber, time.Now()).Scan(
		&cr.Source,
		&cr.Destination,
		&cr.Proxy,
		&cr.OrganizationID,
		&cr.Requested,
		&cr.Expires,
		&cr.CallSID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrCallRequestNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &cr, nil
}

func (d *dal) LookupCallRequest(callSID string) (*models.CallRequest, error) {
	var cr models.CallRequest

	err := d.db.QueryRow(`
		SELECT source, destination, proxy, organization_id, requested, expires, call_sid 
		FROM outgoing_call_request
		WHERE call_sid = ?`, callSID).Scan(
		&cr.Source,
		&cr.Destination,
		&cr.Proxy,
		&cr.OrganizationID,
		&cr.Requested,
		&cr.Expires,
		&cr.CallSID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrCallRequestNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &cr, nil
}

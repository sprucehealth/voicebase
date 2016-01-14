package dal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

type ProvisionedEndpointLookup struct {
	PhoneNumber    *string
	ProvisionedFor *string
}

type OutgoingCallRouteLookup struct {
	CallSID *string
}

type ProxyPhoneNumberOpt int

const (
	PPOUnexpiredOnly ProxyPhoneNumberOpt = 1 << iota
	PPOAll           ProxyPhoneNumberOpt = 0
)

type ProxyPhoneNumberUpdate struct {
	Expires      *time.Time
	LastReserved *time.Time
}

type ProxyPhoneNumberReservationLookup struct {
	ProxyPhoneNumber    *string
	DestinationEntityID *string
}

type ProxyPhoneNumberReservationUpdate struct {
	Expires *time.Time
}

type DAL interface {
	// Transact encapsulates the provided function in a transaction and handles rollback and commit actions
	Transact(func(DAL) error) error

	// LookupProvisionedEndpoint returns a provisioned endpoint.
	LookupProvisionedEndpoint(endpoint string, endpointType models.EndpointType) (*models.ProvisionedEndpoint, error)

	// ProvisionEndpoint provisions the specified endpoint.
	ProvisionEndpoint(ppn *models.ProvisionedEndpoint) error

	// LogCallEvent persists the provided event for operational purposes.
	// TODO: If this data gets noisy, might make sense to log this data into a different
	// database of its own.
	LogCallEvent(e *models.CallEvent) error

	// CreateSentMessage persists a message sent by the excomms service.
	CreateSentMessage(sm *models.SentMessage) error

	// LookupSentMessageByUUID returns any message sent for the specified (UUID, message type) combination.
	LookupSentMessageByUUID(uuid, destination string) (*models.SentMessage, error)

	// CreateCallRequest creates the provided call request.
	CreateCallRequest(*models.CallRequest) error

	// LookupCallRequest returns the call request identified by the call sid.
	LookupCallRequest(callSID string) (*models.CallRequest, error)

	// ProxyPhoneNumbers returns a list of proxy phone numbers based on the provided options.
	ProxyPhoneNumbers(opt ProxyPhoneNumberOpt) ([]*models.ProxyPhoneNumber, error)

	// UpdateProxyPhoneNumber updates the mutable fields for the specified proxy phone number.
	UpdateProxyPhoneNumber(phoneNumber phone.Number, update *ProxyPhoneNumberUpdate) (int64, error)

	// CreateProxyPhoneNumberReservation creates a phone number reservation entry.
	CreateProxyPhoneNumberReservation(model *models.ProxyPhoneNumberReservation) error

	// UpdateActiveProxyPhoneNumberReservation updates an active reservation identified by the proxyPhoneNumber.
	// Note that an active reservation is one that is not expired, and it is enforced at the application layer
	// that there exists only a single active reservation per proxy phone number.
	UpdateActiveProxyPhoneNumberReservation(proxyPhoneNumber phone.Number, update *ProxyPhoneNumberReservationUpdate) (int64, error)

	// ActiveProxyPhoneNumberReservation returns a single reservation for the given lookup based on
	// proxy phone number or destination entity id.
	// Note that an active reservation is one that is not expired, and it is enforced at the application layer
	// that there exists only a single active reservation per proxy phone number.
	ActiveProxyPhoneNumberReservation(lookup *ProxyPhoneNumberReservationLookup) (*models.ProxyPhoneNumberReservation, error)

	// StoreIncomingRawMessage persists the message in the database and returns an ID
	// to identify the message by.
	StoreIncomingRawMessage(rm *rawmsg.Incoming) (uint64, error)

	// IncomingRawMessage returns the raw message based on the id.
	IncomingRawMessage(id uint64) (*rawmsg.Incoming, error)

	// StoreMedia persists a media object
	StoreMedia(media []*models.Media) error

	// LookupMedia looks up media objects based on their IDs
	LookupMedia(ids []uint64) (map[uint64]*models.Media, error)
}

type dal struct {
	db tsql.DB
}

func NewDAL(db *sql.DB) DAL {
	return &dal{
		db: tsql.AsDB(db),
	}
}

var (
	ErrProvisionedEndpointNotFound         = errors.New("provisioned endpoint not found")
	ErrProxyPhoneNumberReservationNotFound = errors.New("phone number reservation not found")
	ErrSentMessageNotFound                 = errors.New("sent message not found")
	ErrIncomingRawMessageNotFound          = errors.New("incoming raw message not found")
)

func (d *dal) LookupProvisionedEndpoint(provisionedFor string, endpointType models.EndpointType) (*models.ProvisionedEndpoint, error) {
	var ppn models.ProvisionedEndpoint

	err := d.db.QueryRow(`
		SELECT endpoint, endpoint_type, provisioned_for, created 
		FROM provisioned_endpoint
		WHERE provisioned_for = ?
		AND endpoint_type = ?`, provisionedFor, endpointType).Scan(
		&ppn.Endpoint,
		&ppn.EndpointType,
		&ppn.ProvisionedFor,
		&ppn.Provisioned)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrProvisionedEndpointNotFound)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &ppn, nil
}

func (d *dal) ProvisionEndpoint(ppn *models.ProvisionedEndpoint) error {
	if ppn == nil {
		return nil
	}

	_, err := d.db.Exec(`
		INSERT INTO provisioned_endpoint (endpoint, endpoint_type, provisioned_for) VALUES (?,?,?)`, ppn.Endpoint, ppn.EndpointType, ppn.ProvisionedFor)
	return errors.Trace(err)
}

func (d *dal) LogCallEvent(e *models.CallEvent) error {
	if e == nil {
		return nil
	}

	data, err := json.Marshal(e)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = d.db.Exec(`
		INSERT INTO twilio_call_event (source, destination, data, event)
		VALUES (?,?,?,?)`, e.Source, e.Destination, data, e.Type)

	return errors.Trace(err)
}

func (d *dal) CreateSentMessage(sm *models.SentMessage) error {
	if sm.ID == 0 {
		id, err := idgen.NewID()
		if err != nil {
			return errors.Trace(err)
		}
		sm.ID = id
	}

	data, err := sm.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	_, err = d.db.Exec(`
		INSERT INTO sent_message (id, uuid, type, destination, data) VALUES (?, ?, ?, ?, ?)
		`, sm.ID, sm.UUID, sm.Type, sm.Destination, data)
	return errors.Trace(err)
}

func (d *dal) LookupSentMessageByUUID(uuid, destination string) (*models.SentMessage, error) {
	var data []byte
	if err := d.db.QueryRow(`
		SELECT data
		FROM sent_message
		WHERE uuid = ?
		AND destination = ?`, uuid, destination).Scan(
		&data); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrSentMessageNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	var sm models.SentMessage
	if err := sm.Unmarshal(data); err != nil {
		return nil, errors.Trace(err)
	}

	return &sm, nil
}

func (d *dal) CreateCallRequest(cr *models.CallRequest) error {
	if cr == nil {
		return nil
	}

	_, err := d.db.Exec(`
		INSERT INTO outgoing_call_request (source, destination, proxy, organization_id, requested, call_sid)
		VALUES (?,?,?,?,?,?)`, cr.Source, cr.Destination, cr.Proxy, cr.OrganizationID, cr.Requested, cr.CallSID)
	return errors.Trace(err)
}

var ErrCallRequestNotFound = errors.New("call request not found")

func (d *dal) LookupCallRequest(callSID string) (*models.CallRequest, error) {
	var cr models.CallRequest

	err := d.db.QueryRow(`
		SELECT source, destination, proxy, organization_id, requested, call_sid 
		FROM outgoing_call_request
		WHERE call_sid = ?`, callSID).Scan(
		&cr.Source,
		&cr.Destination,
		&cr.Proxy,
		&cr.OrganizationID,
		&cr.Requested,
		&cr.CallSID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrCallRequestNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &cr, nil
}

func (d *dal) ProxyPhoneNumbers(opt ProxyPhoneNumberOpt) ([]*models.ProxyPhoneNumber, error) {
	var rows *sql.Rows
	var err error
	if opt&PPOUnexpiredOnly == PPOUnexpiredOnly {
		rows, err = d.db.Query(`
			SELECT phone_number, expires, last_reserved_time
			FROM proxy_phone_number
			WHERE (expires IS NULL) OR (expires < ?)
			FOR UPDATE
		`, time.Now())
		if err != nil {
			return nil, errors.Trace(err)
		}
	} else {
		rows, err = d.db.Query(`
			SELECT phone_number, expires, last_reserved_time
			FROM proxy_phone_number
			FOR UPDATE
		`)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	defer rows.Close()

	var phoneNumbers []*models.ProxyPhoneNumber
	for rows.Next() {
		var ppn models.ProxyPhoneNumber
		if err := rows.Scan(&ppn.PhoneNumber, &ppn.Expires, &ppn.LastReserved); err != nil {
			return nil, errors.Trace(err)
		}
		phoneNumbers = append(phoneNumbers, &ppn)
	}

	return phoneNumbers, errors.Trace(rows.Err())
}

func (d *dal) UpdateProxyPhoneNumber(phoneNumber phone.Number, update *ProxyPhoneNumberUpdate) (int64, error) {

	if update == nil {
		return 0, nil
	}

	vars := dbutil.MySQLVarArgs()

	if update.Expires != nil {
		vars.Append("expires", *update.Expires)
	}
	if update.LastReserved != nil {
		vars.Append("last_reserved_time", *update.LastReserved)
	}

	if vars.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(`
		UPDATE proxy_phone_number
		SET `+vars.ColumnsForUpdate()+`
		WHERE phone_number = ?
		`, append(vars.Values(), phoneNumber)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsAffected, nil
}

func (d *dal) CreateProxyPhoneNumberReservation(model *models.ProxyPhoneNumberReservation) error {
	_, err := d.db.Exec(`
		INSERT INTO proxy_phone_number_reservation (phone_number, destination_entity_id, owner_entity_id, organization_id, expires)
		VALUES (?, ?, ?, ?, ?)`, model.PhoneNumber, model.DestinationEntityID, model.OwnerEntityID, model.OrganizationID, model.Expires)
	return errors.Trace(err)
}

func (d *dal) ActiveProxyPhoneNumberReservation(lookup *ProxyPhoneNumberReservationLookup) (*models.ProxyPhoneNumberReservation, error) {
	var where []string
	var vals []interface{}

	if lookup.DestinationEntityID != nil {
		where = append(where, "destination_entity_id = ?")
		vals = append(vals, *lookup.DestinationEntityID)
	}
	if lookup.ProxyPhoneNumber != nil {
		where = append(where, "phone_number = ?")
		vals = append(vals, *lookup.ProxyPhoneNumber)
	}

	if lookup.DestinationEntityID == nil && lookup.ProxyPhoneNumber == nil {
		return nil, errors.Trace(fmt.Errorf("destination_entity_id or phone_number required"))
	}

	var ppnr models.ProxyPhoneNumberReservation
	err := d.db.QueryRow(`
		SELECT phone_number, destination_entity_id, owner_entity_id, organization_id, expires
		FROM proxy_phone_number_reservation
		WHERE `+strings.Join(where, " AND ")+`
		AND expires > ?`, append(vals, time.Now())...).Scan(
		&ppnr.PhoneNumber,
		&ppnr.DestinationEntityID,
		&ppnr.OwnerEntityID,
		&ppnr.OrganizationID,
		&ppnr.Expires)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrProxyPhoneNumberReservationNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &ppnr, nil
}

func (d *dal) UpdateActiveProxyPhoneNumberReservation(proxyPhoneNumber phone.Number, update *ProxyPhoneNumberReservationUpdate) (int64, error) {
	if update == nil {
		return 0, nil
	}

	vars := dbutil.MySQLVarArgs()

	if update.Expires != nil {
		vars.Append("expires", *update.Expires)
	}

	if vars.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(`
		UPDATE proxy_phone_number_reservation
		SET `+vars.ColumnsForUpdate()+`
		WHERE phone_number = ?
		AND expires > ?
		`, append(vars.Values(), proxyPhoneNumber, time.Now())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsAffected, nil
}

func (d *dal) Transact(trans func(DAL) error) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	tdal := &dal{
		db: tsql.AsSafeTx(tx),
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			errString := fmt.Sprintf("Encountered panic during transaction execution: %v", r)
			golog.Errorf(errString)
			err = errors.Trace(errors.New(errString))
		}
	}()

	if err := trans(tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return errors.Trace(tx.Commit())
}

func (d *dal) StoreIncomingRawMessage(rm *rawmsg.Incoming) (uint64, error) {
	id, err := idgen.NewID()
	if err != nil {
		return 0, errors.Trace(err)
	}

	data, err := rm.Marshal()
	if err != nil {
		return 0, errors.Trace(err)
	}

	_, err = d.db.Exec(`
		INSERT INTO incoming_raw_message (id, type, data) VALUES (?,?,?)`, id, rm.Type.String(), data)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return id, nil
}

func (d *dal) IncomingRawMessage(id uint64) (*rawmsg.Incoming, error) {
	var data []byte
	if err := d.db.QueryRow(`
		SELECT data 
		FROM incoming_raw_message
		WHERE id = ?`, id).Scan(&data); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrIncomingRawMessageNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	var rm rawmsg.Incoming
	if err := rm.Unmarshal(data); err != nil {
		return nil, errors.Trace(err)
	}
	return &rm, nil
}

func (d *dal) StoreMedia(media []*models.Media) error {
	if len(media) == 0 {
		return nil
	}

	multiInsert := dbutil.MySQLMultiInsert(len(media))
	for _, m := range media {
		if m.ID == 0 {
			return errors.Trace(fmt.Errorf("id required for media object"))
		}

		multiInsert.Append(m.ID, m.Type, m.URL, m.Name)
	}

	_, err := d.db.Exec(`INSERT INTO media (id, type, url, name) VALUES `+multiInsert.Query(), multiInsert.Values()...)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LookupMedia looks up media objects based on their IDs
func (d *dal) LookupMedia(ids []uint64) (map[uint64]*models.Media, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, type, url, name
		FROM media 
		WHERE id in (`+dbutil.MySQLArgs(len(ids))+`)`, dbutil.AppendUint64sToInterfaceSlice(nil, ids)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	media := make(map[uint64]*models.Media)
	for rows.Next() {
		var m models.Media
		if err := rows.Scan(
			&m.ID,
			&m.Type,
			&m.URL,
			&m.Name); err != nil {
			return nil, errors.Trace(err)
		}
		media[m.ID] = &m
	}

	return media, errors.Trace(rows.Err())
}

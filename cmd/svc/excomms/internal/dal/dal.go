package dal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"golang.org/x/net/context"
)

// QueryOption is an optional that can be provided to a DAL function
type QueryOption int

const (
	// ForUpdate locks the queried rows for update
	ForUpdate QueryOption = iota + 1
)

type queryOptions []QueryOption

func (qos queryOptions) Has(opt QueryOption) bool {
	for _, o := range qos {
		if o == opt {
			return true
		}
	}
	return false
}

type IPCallUpdate struct {
	Pending       *bool
	ConnectedTime *time.Time
}

type IPCallParticipantUpdate struct {
	State       *models.IPCallState
	NetworkType *models.NetworkType
}

type ProvisionedEndpointLookup struct {
	PhoneNumber    *string
	ProvisionedFor *string
}

type OutgoingCallRouteLookup struct {
	CallSID *string
}

type ProxyPhoneNumberReservationUpdate struct {
	Expires *time.Time
}

type ProvisionedEndpointUpdate struct {
	Deprovisioned          *bool
	DeprovisionedReason    *string
	DeprovisionedTimestamp *time.Time
}

type IncomingCallUpdate struct {
	Afterhours *bool
	Urgent     *bool
}

type DAL interface {
	// Transact encapsulates the provided function in a transaction and handles rollback and commit actions
	Transact(func(DAL) error) error

	// LookupProvisionedEndpoint returns a provisioned endpoint.
	LookupProvisionedEndpoint(endpoint string, endpointType models.EndpointType) (*models.ProvisionedEndpoint, error)

	// ProvisionEndpoint provisions the specified endpoint.
	ProvisionEndpoint(ppn *models.ProvisionedEndpoint) error

	// UpdateProvisionedEndpoint updates the mutable and requested fields of the provisioned endpoint row
	UpdateProvisionedEndpoint(endpoint string, endpointType models.EndpointType, update *ProvisionedEndpointUpdate) (int64, error)

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

	// CreateIncomingCall creates an entry for the incoming call
	CreateIncomingCall(*models.IncomingCall) error

	// LookupIncomingCall identifies an incoming call by the SID
	LookupIncomingCall(sid string) (*models.IncomingCall, error)

	// UpdateIncomingCall allows updating of the mutable properties of the incoming call
	UpdateIncomingCall(sid string, update *IncomingCallUpdate) (int64, error)

	// AvailableProxyPhoneNumbers returns a list of proxy phone numbers available for reservation for a given originatingNumber.
	AvailableProxyPhoneNumbers(originatingPhoneNumber phone.Number) ([]*models.ProxyPhoneNumber, error)

	// CreateProxyPhoneNumberReservation creates a phone number reservation entry.
	CreateProxyPhoneNumberReservation(model *models.ProxyPhoneNumberReservation) error

	// UpdateActiveProxyPhoneNumberReservation updates an active reservation identified by the proxyPhoneNumber.
	// Note that an active reservation is one that is not expired, and it is enforced at the application layer
	// that there exists only a single active reservation per proxy phone number.
	UpdateActiveProxyPhoneNumberReservation(originatingPhoneNumber phone.Number, destinationPhoneNumber, proxyPhoneNumber *phone.Number, update *ProxyPhoneNumberReservationUpdate) (int64, error)

	// ActiveProxyPhoneNumberReservation returns an unexpired (aka "active") reservation uniquely identified by the (originatingPhoneNumber, destinationPhoneNumber) or
	// (originatingPhoneNumber, proxyPhoneNumber) pair.
	ActiveProxyPhoneNumberReservation(originatingPhoneNumber, destinationPhoneNumber, proxyPhoneNumber *phone.Number) (*models.ProxyPhoneNumberReservation, error)

	// SetCurrentOriginatingNumber sets the provided originaing number for the entityID.
	SetCurrentOriginatingNumber(phoneNumber phone.Number, entityID, deviceID string) error

	// OriginatingNumber returns the current originating number for the given entityID.
	CurrentOriginatingNumber(entityID, deviceID string) (phone.Number, error)

	// StoreIncomingRawMessage persists the message in the database and returns an ID
	// to identify the message by.
	StoreIncomingRawMessage(rm *rawmsg.Incoming) (uint64, error)

	// IncomingRawMessage returns the raw message based on the id.
	IncomingRawMessage(id uint64) (*rawmsg.Incoming, error)

	// StoreMedia persists a media object
	StoreMedia(media []*models.Media) error

	// LookupMedia looks up media objects based on their IDs
	LookupMedia(ids []string) (map[string]*models.Media, error)

	// CreateDeletedResource creates an entry for a deleted resource
	CreateDeletedResource(resource, resourceID string) error

	// CreateIPCall creates an IP call along with the list of participants
	CreateIPCall(ctx context.Context, call *models.IPCall) error

	// IPCall returns an IPCall by ID
	IPCall(ctx context.Context, id models.IPCallID, opts ...QueryOption) (*models.IPCall, error)

	// PendingIPCallsForAccount returns the pending IP calls for an account
	PendingIPCallsForAccount(ctx context.Context, accountID string) ([]*models.IPCall, error)

	// UpdatePendingCall updates an IP call
	UpdateIPCall(ctx context.Context, callID models.IPCallID, update *IPCallUpdate) error

	// UpdateIPCallParticipant updates a participant of an IP call
	UpdateIPCallParticipant(ctx context.Context, callID models.IPCallID, accountID string, update *IPCallParticipantUpdate) error
}

type dal struct {
	db  tsql.DB
	clk clock.Clock
}

// New returns a new initialized DAL that uses an SQL database
func New(db *sql.DB, clk clock.Clock) DAL {
	return &dal{
		db:  tsql.AsDB(db),
		clk: clk,
	}
}

var (
	ErrProvisionedEndpointNotFound         = errors.New("provisioned endpoint not found")
	ErrProxyPhoneNumberReservationNotFound = errors.New("phone number reservation not found")
	ErrSentMessageNotFound                 = errors.New("sent message not found")
	ErrIncomingRawMessageNotFound          = errors.New("incoming raw message not found")
	ErrOriginatingNumberNotFound           = errors.New("originating number not found")
	ErrIncomingCallNotFound                = errors.New("incoming_call not found")
	ErrCallRequestNotFound                 = errors.New("call request not found")
	ErrIPCallNotFound                      = errors.New("ipcall not found")
)

func (d *dal) LookupProvisionedEndpoint(provisionedFor string, endpointType models.EndpointType) (*models.ProvisionedEndpoint, error) {
	var ppn models.ProvisionedEndpoint

	err := d.db.QueryRow(`
		SELECT endpoint, endpoint_type, provisioned_for, created, deprovisioned, deprovisioned_timestamp, deprovisioned_reason
		FROM provisioned_endpoint
		WHERE provisioned_for = ?
		AND endpoint_type = ?
		AND deprovisioned = false`, provisionedFor, endpointType).Scan(
		&ppn.Endpoint,
		&ppn.EndpointType,
		&ppn.ProvisionedFor,
		&ppn.Provisioned,
		&ppn.Deprovisioned,
		&ppn.DeprovisionedTimestamp,
		&ppn.DeprovisionedReason)
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

func (d *dal) UpdateProvisionedEndpoint(endpoint string, endpointType models.EndpointType, update *ProvisionedEndpointUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Deprovisioned != nil {
		args.Append("deprovisioned", *update.Deprovisioned)
	}
	if update.DeprovisionedReason != nil {
		args.Append("deprovisioned_reason", *update.DeprovisionedReason)
	}
	if update.DeprovisionedTimestamp != nil {
		args.Append("deprovisioned_timestamp", *update.DeprovisionedTimestamp)
	}

	if args == nil || args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(`
		UPDATE provisioned_endpoint
		SET `+args.ColumnsForUpdate()+`
		WHERE endpoint = ? AND endpoint_type = ?`, append(args.Values(), endpoint, endpointType)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsUpdated, nil
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
		INSERT INTO outgoing_call_request (source, destination, proxy, organization_id, requested, call_sid, caller_entity_id, callee_entity_id)
		VALUES (?,?,?,?,?,?,?,?)`, cr.Source, cr.Destination, cr.Proxy, cr.OrganizationID, cr.Requested, cr.CallSID, cr.CallerEntityID, cr.CalleeEntityID)
	return errors.Trace(err)
}

func (d *dal) LookupCallRequest(callSID string) (*models.CallRequest, error) {
	var cr models.CallRequest

	err := d.db.QueryRow(`
		SELECT source, destination, proxy, organization_id, requested, call_sid, caller_entity_id, callee_entity_id
		FROM outgoing_call_request
		WHERE call_sid = ?`, callSID).Scan(
		&cr.Source,
		&cr.Destination,
		&cr.Proxy,
		&cr.OrganizationID,
		&cr.Requested,
		&cr.CallSID,
		&cr.CallerEntityID,
		&cr.CalleeEntityID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrCallRequestNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &cr, nil
}

func (d *dal) AvailableProxyPhoneNumbers(originatingPhoneNumber phone.Number) ([]*models.ProxyPhoneNumber, error) {

	// first, get the authoritative list of proxy phone numbers
	// even though we are doing a full table scan this table should be fairly small.
	rows, err := d.db.Query(`
		SELECT phone_number
		FROM proxy_phone_number`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var proxyPhoneNumbers []*models.ProxyPhoneNumber
	proxyPhoneNumbersMap := make(map[string]*models.ProxyPhoneNumber)
	for rows.Next() {
		var pn models.ProxyPhoneNumber
		if err := rows.Scan(&pn.PhoneNumber); err != nil {
			return nil, errors.Trace(err)
		}
		proxyPhoneNumbers = append(proxyPhoneNumbers, &pn)
		proxyPhoneNumbersMap[pn.PhoneNumber.String()] = &pn
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}

	// then, get the latest reservation that exists
	// for each proxy phone number to determine which of them are
	// currently reserved and exclude that from the list of phone numbers
	// to return
	rows2, err := d.db.Query(`
		SELECT proxy_phone_number, MAX(expires)
		FROM proxy_phone_number_reservation
		WHERE originating_phone_number = ?
		GROUP BY proxy_phone_number`, originatingPhoneNumber)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows2.Close()

	availableProxyNumbers := make([]*models.ProxyPhoneNumber, 0, len(proxyPhoneNumbers))
	for rows2.Next() {
		var ppn models.ProxyPhoneNumber
		if err := rows2.Scan(&ppn.PhoneNumber, &ppn.Expires); err != nil {
			return nil, errors.Trace(err)
		}

		if ppn.Expires.Before(time.Now()) {
			// ensure that the phone number can still be used as a proxy phone number
			if _, ok := proxyPhoneNumbersMap[ppn.PhoneNumber.String()]; ok {
				availableProxyNumbers = append(availableProxyNumbers, &ppn)
			}
		}

		// delete any proxy phone number from the map after it has been considered
		delete(proxyPhoneNumbersMap, ppn.PhoneNumber.String())
	}

	// whatever remains in the map are numbers that have never been reserved and can now be considered
	for _, pn := range proxyPhoneNumbersMap {
		availableProxyNumbers = append(availableProxyNumbers, pn)
	}

	return availableProxyNumbers, errors.Trace(rows2.Err())
}

func (d *dal) CreateProxyPhoneNumberReservation(model *models.ProxyPhoneNumberReservation) error {
	_, err := d.db.Exec(`
		INSERT INTO proxy_phone_number_reservation (proxy_phone_number, originating_phone_number, destination_phone_number, destination_entity_id, owner_entity_id, organization_id, expires)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, model.ProxyPhoneNumber, model.OriginatingPhoneNumber, model.DestinationPhoneNumber, model.DestinationEntityID, model.OwnerEntityID, model.OrganizationID, model.Expires)
	return errors.Trace(err)
}

func (d *dal) ActiveProxyPhoneNumberReservation(originatingPhoneNumber, destinationPhoneNumber, proxyPhoneNumber *phone.Number) (*models.ProxyPhoneNumberReservation, error) {

	where := make([]string, 0, 3)
	vals := make([]interface{}, 0, 3)

	if originatingPhoneNumber != nil {
		where = append(where, "originating_phone_number = ?")
		vals = append(vals, *originatingPhoneNumber)
	}

	if destinationPhoneNumber != nil {
		where = append(where, "destination_phone_number = ?")
		vals = append(vals, *destinationPhoneNumber)
	}
	if proxyPhoneNumber != nil {
		where = append(where, "proxy_phone_number = ?")
		vals = append(vals, *proxyPhoneNumber)
	}

	if len(where) == 0 {
		return nil, errors.Trace(fmt.Errorf("either destination_phone_numer or proxy_phone_number must be specified"))
	}

	var ppnr models.ProxyPhoneNumberReservation
	err := d.db.QueryRow(`
		SELECT proxy_phone_number, originating_phone_number, destination_phone_number, destination_entity_id, owner_entity_id, organization_id, created, expires
		FROM proxy_phone_number_reservation
		WHERE `+strings.Join(where, " AND ")+`
		AND expires > ?`, append(vals, time.Now())...).Scan(
		&ppnr.ProxyPhoneNumber,
		&ppnr.OriginatingPhoneNumber,
		&ppnr.DestinationPhoneNumber,
		&ppnr.DestinationEntityID,
		&ppnr.OwnerEntityID,
		&ppnr.OrganizationID,
		&ppnr.Created,
		&ppnr.Expires)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrProxyPhoneNumberReservationNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &ppnr, nil
}

func (d *dal) UpdateActiveProxyPhoneNumberReservation(originatingPhoneNumber phone.Number, destinationPhoneNumber, proxyPhoneNumber *phone.Number, update *ProxyPhoneNumberReservationUpdate) (int64, error) {
	if update == nil {
		return 0, nil
	}

	where := make([]string, 0, 2)
	vals := make([]interface{}, 0, 3)

	where = append(where, "originating_phone_number = ?")
	vals = append(vals, originatingPhoneNumber)
	if destinationPhoneNumber != nil {
		where = append(where, "destination_phone_number = ?")
		vals = append(vals, *destinationPhoneNumber)
	}
	if proxyPhoneNumber != nil {
		where = append(where, "proxy_phone_number = ?")
		vals = append(vals, *proxyPhoneNumber)
	}
	vals = append(vals, time.Now())

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
		WHERE `+strings.Join(where, " AND ")+`
		AND expires > ?
		`, append(vars.Values(), vals...)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsAffected, nil
}

func (d *dal) SetCurrentOriginatingNumber(phoneNumber phone.Number, entityID, deviceID string) error {
	_, err := d.db.Exec(`REPLACE INTO originating_phone_number (phone_number, entity_id, device_id) VALUES (?, ?, ?)`, phoneNumber, entityID, deviceID)
	return errors.Trace(err)
}

func (d *dal) CurrentOriginatingNumber(entityID, deviceID string) (phone.Number, error) {
	var phoneNumber phone.Number
	err := d.db.QueryRow(`
		SELECT phone_number
		FROM originating_phone_number
		WHERE entity_id = ? AND device_id = ?`, entityID, deviceID).Scan(&phoneNumber)
	if err == sql.ErrNoRows {
		return phone.Number(""), errors.Trace(ErrOriginatingNumberNotFound)
	}
	return phoneNumber, errors.Trace(err)
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
	if rm.ID == 0 {
		id, err := idgen.NewID()
		if err != nil {
			return 0, errors.Trace(err)
		}
		rm.ID = id
	}

	data, err := rm.Marshal()
	if err != nil {
		return 0, errors.Trace(err)
	}

	_, err = d.db.Exec(`
		INSERT INTO incoming_raw_message (id, type, data) VALUES (?,?,?)
		ON DUPLICATE KEY UPDATE data = ?`, rm.ID, rm.Type.String(), data, data)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rm.ID, nil
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
		if m.ID == "" {
			return errors.Trace(fmt.Errorf("id required for media object"))
		}

		multiInsert.Append(m.ID, m.Type, m.Location, m.Name)
		golog.Debugf("Inserting media %+v", m)
	}

	_, err := d.db.Exec(`INSERT INTO media (id, type, url, name) VALUES `+multiInsert.Query(), multiInsert.Values()...)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LookupMedia looks up media objects based on their IDs
func (d *dal) LookupMedia(ids []string) (map[string]*models.Media, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, type, url, name
		FROM media
		WHERE id in (`+dbutil.MySQLArgs(len(ids))+`)`, dbutil.AppendStringsToInterfaceSlice(nil, ids)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	media := make(map[string]*models.Media)
	for rows.Next() {
		var m models.Media
		if err := rows.Scan(
			&m.ID,
			&m.Type,
			&m.Location,
			&m.Name); err != nil {
			return nil, errors.Trace(err)
		}
		media[m.ID] = &m
	}

	return media, errors.Trace(rows.Err())
}

func (d *dal) CreateIncomingCall(ic *models.IncomingCall) error {
	_, err := d.db.Exec(`REPLACE INTO incoming_call (call_sid, source, destination, organization_id, afterhours, urgent) VALUES (?,?,?,?,?,?)`, ic.CallSID, ic.Source, ic.Destination, ic.OrganizationID, ic.AfterHours, ic.Urgent)
	return errors.Trace(err)
}

func (d *dal) LookupIncomingCall(sid string) (*models.IncomingCall, error) {
	var ic models.IncomingCall
	if err := d.db.QueryRow(`
		SELECT call_sid, source, destination, organization_id, afterhours, urgent
		FROM incoming_call
		WHERE call_sid = ?`, sid).Scan(
		&ic.CallSID,
		&ic.Source,
		&ic.Destination,
		&ic.OrganizationID,
		&ic.AfterHours,
		&ic.Urgent); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrIncomingCallNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &ic, nil
}

func (d *dal) UpdateIncomingCall(sid string, update *IncomingCallUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Afterhours != nil {
		args.Append("afterhours", *update.Afterhours)
	}
	if update.Urgent != nil {
		args.Append("urgent", *update.Urgent)
	}

	if args == nil || args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(`
		UPDATE incoming_call
		SET `+args.ColumnsForUpdate()+`
		WHERE call_sid = ?`, append(args.Values(), sid)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsUpdated, nil
}

func (d *dal) CreateDeletedResource(resource, resourceID string) error {
	_, err := d.db.Exec(`INSERT INTO deleted_resource (resource, resource_id) VALUES (?,?)`, resource, resourceID)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

package dal

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"context"

	"database/sql/driver"

	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	modellib "github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

// QueryOption represents an option available to a query
type QueryOption int

const (
	// ForUpdate represents the FOR UPDATE to be appended to a query
	ForUpdate QueryOption = 1 << iota
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

// ErrNotFound is returned when an item is not found
var ErrNotFound = errors.New("payments/dal: item not found")

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	Transact(ctx context.Context, trans func(ctx context.Context, dal DAL) error) (err error)
	DeleteVendorAccount(ctx context.Context, id VendorAccountID) (int64, error)
	EntityVendorAccounts(ctx context.Context, entityID string, opts ...QueryOption) ([]*VendorAccount, error)
	InsertVendorAccount(ctx context.Context, model *VendorAccount) (VendorAccountID, error)
	UpdateVendorAccount(ctx context.Context, id VendorAccountID, update *VendorAccountUpdate) error
	VendorAccount(ctx context.Context, id VendorAccountID, opts ...QueryOption) (*VendorAccount, error)
	VendorAccountsInState(ctx context.Context, lifecycle VendorAccountLifecycle, changeState VendorAccountChangeState, limit int64, opts ...QueryOption) ([]*VendorAccount, error)
}

type dal struct {
	db tsql.DB
}

// New returns an initialized instance of dal
func New(db *sql.DB) DAL {
	return &dal{db: tsql.AsDB(db)}
}

// Transact encapsulated the provided function in a transaction and handles rollback and commit actions
func (d *dal) Transact(ctx context.Context, trans func(ctx context.Context, dal DAL) error) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	tdal := &dal{
		db: tsql.AsSafeTx(tx),
	}
	// Recover from any inner panics that happened and close the transaction
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = errors.Trace(fmt.Errorf("Encountered panic during transaction execution: %v", r))
		}
	}()
	if err := trans(ctx, tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

// VendorAccountIDPrefix represents the string that is attached to the beginning of these identifiers
const VendorAccountIDPrefix = "vendorAccount_"

// NewVendorAccountID returns a new VendorAccountID.
func NewVendorAccountID() (VendorAccountID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return VendorAccountID{}, errors.Trace(err)
	}
	return VendorAccountID{
		modellib.ObjectID{
			Prefix:  VendorAccountIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyVendorAccountID returns an empty initialized ID
func EmptyVendorAccountID() VendorAccountID {
	return VendorAccountID{
		modellib.ObjectID{
			Prefix:  VendorAccountIDPrefix,
			IsValid: false,
		},
	}
}

// ParseVendorAccountID transforms an VendorAccountID from it's string representation into the actual ID value
func ParseVendorAccountID(s string) (VendorAccountID, error) {
	id := EmptyVendorAccountID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// VendorAccountID is the ID for a VendorAccountID object
type VendorAccountID struct {
	modellib.ObjectID
}

// VendorAccountLifecycle represents the type associated with the status column of the vendor_account table
type VendorAccountLifecycle string

const (
	// VendorAccountLifecycleConnected represents the CONNECTED state of the status field on a vendor_account record
	VendorAccountLifecycleConnected VendorAccountLifecycle = "CONNECTED"
	// VendorAccountLifecycleDisconnected represents the DISCONNECTED state of the status field on a vendor_account record
	VendorAccountLifecycleDisconnected VendorAccountLifecycle = "DISCONNECTED"
)

// ParseVendorAccountLifecycle converts a string into the corresponding enum value
func ParseVendorAccountLifecycle(s string) (VendorAccountLifecycle, error) {
	switch t := VendorAccountLifecycle(strings.ToUpper(s)); t {
	case VendorAccountLifecycleConnected, VendorAccountLifecycleDisconnected:
		return t, nil
	}
	return VendorAccountLifecycle(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t VendorAccountLifecycle) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t VendorAccountLifecycle) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of VendorAccountLifecycle from a database conforming to the sql.Scanner interface
func (t *VendorAccountLifecycle) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseVendorAccountLifecycle(ts)
	case []byte:
		*t, err = ParseVendorAccountLifecycle(string(ts))
	}
	return errors.Trace(err)
}

// VendorAccountChangeState represents the type associated with the change_state column of the vendor_account table
type VendorAccountChangeState string

const (
	// VendorAccountChangeStateNone represents the NONE state of the change_state field on a vendor_account record
	VendorAccountChangeStateNone VendorAccountChangeState = "NONE"
	// VendorAccountChangeStatePending represents the PENDING state of the change_state field on a vendor_account record
	VendorAccountChangeStatePending VendorAccountChangeState = "PENDING"
)

// ParseVendorAccountChangeState converts a string into the corresponding enum value
func ParseVendorAccountChangeState(s string) (VendorAccountChangeState, error) {
	switch t := VendorAccountChangeState(strings.ToUpper(s)); t {
	case VendorAccountChangeStateNone, VendorAccountChangeStatePending:
		return t, nil
	}
	return VendorAccountChangeState(""), errors.Trace(fmt.Errorf("Unknown change state:%s", s))
}

func (t VendorAccountChangeState) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t VendorAccountChangeState) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of VendorAccountChangeState from a database conforming to the sql.Scanner interface
func (t *VendorAccountChangeState) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseVendorAccountChangeState(ts)
	case []byte:
		*t, err = ParseVendorAccountChangeState(string(ts))
	}
	return errors.Trace(err)
}

// VendorAccountAccountType represents the type associated with the account_type column of the vendor_account table
type VendorAccountAccountType string

const (
	// VendorAccountAccountTypeStripe represents the STRIPE state of the account_type field on a vendor_account record
	VendorAccountAccountTypeStripe VendorAccountAccountType = "STRIPE"
)

// ParseVendorAccountAccountType converts a string into the correcponding enum value
func ParseVendorAccountAccountType(s string) (VendorAccountAccountType, error) {
	switch t := VendorAccountAccountType(strings.ToUpper(s)); t {
	case VendorAccountAccountTypeStripe:
		return t, nil
	}
	return VendorAccountAccountType(""), errors.Trace(fmt.Errorf("Unknown account_type:%s", s))
}

func (t VendorAccountAccountType) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t VendorAccountAccountType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of VendorAccountAccountType from a database conforming to the sql.Scanner interface
func (t *VendorAccountAccountType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseVendorAccountAccountType(ts)
	case []byte:
		*t, err = ParseVendorAccountAccountType(string(ts))
	}
	return errors.Trace(err)
}

// VendorAccount represents a vendor_account record
type VendorAccount struct {
	Scope              string
	Lifecycle          VendorAccountLifecycle
	ChangeState        VendorAccountChangeState
	Modified           time.Time
	AccessToken        string
	PublishableKey     string
	RefreshToken       string
	ConnectedAccountID string
	Live               bool
	AccountType        VendorAccountAccountType
	Created            time.Time
	ID                 VendorAccountID
	EntityID           string
}

// Validate asserts that the object is well formed
func (m *VendorAccount) Validate() error {
	if m.Lifecycle == "" {
		return errors.New("Lifecycle cannot be empty")
	}
	if m.ChangeState == "" {
		return errors.New("ChangeState cannot be empty")
	}
	return nil
}

// VendorAccountUpdate represents the mutable aspects of a vendor_account record
type VendorAccountUpdate struct {
	Lifecycle   VendorAccountLifecycle
	ChangeState VendorAccountChangeState
}

// Validate asserts that the object is well formed
func (m *VendorAccountUpdate) Validate() error {
	if m.Lifecycle == "" {
		return errors.New("Lifecycle cannot be empty")
	}
	if m.ChangeState == "" {
		return errors.New("ChangeState cannot be empty")
	}
	return nil
}

// InsertVendorAccount inserts a vendor_account record
func (d *dal) InsertVendorAccount(ctx context.Context, model *VendorAccount) (VendorAccountID, error) {
	if !model.ID.IsValid {
		id, err := NewVendorAccountID()
		if err != nil {
			return EmptyVendorAccountID(), errors.Trace(err)
		}
		model.ID = id
	}
	if err := model.Validate(); err != nil {
		return EmptyVendorAccountID(), errors.Trace(err)
	}
	_, err := d.db.Exec(
		`INSERT INTO vendor_account
          (access_token, refresh_token, publishable_key, connected_account_id, live, scope, lifecycle, change_state, id, entity_id, account_type)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, model.AccessToken, model.RefreshToken, model.PublishableKey, model.ConnectedAccountID, model.Live, model.Scope, model.Lifecycle, model.ChangeState, model.ID, model.EntityID, model.AccountType)
	if err != nil {
		return EmptyVendorAccountID(), errors.Trace(err)
	}

	return model.ID, nil
}

// VendorAccount retrieves a vendor_account record
func (d *dal) VendorAccount(ctx context.Context, id VendorAccountID, opts ...QueryOption) (*VendorAccount, error) {
	q := selectVendorAccount + ` WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, id.Val)
	model, err := scanVendorAccount(row, id.String())
	return model, errors.Trace(err)
}

// VendorAccountsInState retrieves a random set of vendor_account records with the corresponding state
func (d *dal) VendorAccountsInState(ctx context.Context, lifecycle VendorAccountLifecycle, changeState VendorAccountChangeState, limit int64, opts ...QueryOption) ([]*VendorAccount, error) {
	q := selectVendorAccount + ` WHERE lifecycle = ? AND change_state = ?`
	q += ` ORDER BY RAND() LIMIT ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, lifecycle, changeState, limit)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var vendorAccounts []*VendorAccount
	for rows.Next() {
		va, err := scanVendorAccount(rows, fmt.Sprintf("lifecycle: %s - change_state: %s - limit: %d", lifecycle, changeState, limit))
		if err != nil {
			return nil, errors.Trace(err)
		}
		vendorAccounts = append(vendorAccounts, va)
	}
	return vendorAccounts, errors.Trace(rows.Err())
}

// EntityVendorAccounts retrieves a set of vendor_account records for the provided entity id
func (d *dal) EntityVendorAccounts(ctx context.Context, entityID string, opts ...QueryOption) ([]*VendorAccount, error) {
	q := selectVendorAccount + ` WHERE entity_id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var vendorAccounts []*VendorAccount
	for rows.Next() {
		va, err := scanVendorAccount(rows, entityID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		vendorAccounts = append(vendorAccounts, va)
	}
	return vendorAccounts, errors.Trace(rows.Err())
}

// UpdateVendorAccount updates the mutable aspects of a vendor_account record
func (d *dal) UpdateVendorAccount(ctx context.Context, id VendorAccountID, update *VendorAccountUpdate) error {
	if update == nil {
		return nil
	}
	if err := update.Validate(); err != nil {
		return errors.Trace(err)
	}

	args := dbutil.MySQLVarArgs()
	args.Append("lifecycle", update.Lifecycle)
	args.Append("change_state", update.ChangeState)
	if args.IsEmpty() {
		return nil
	}

	_, err := d.db.Exec(`UPDATE vendor_account SET `+args.ColumnsForUpdate()+` WHERE id = ?`,
		append(args.Values(), id)...)
	return errors.Trace(err)
}

// DeleteVendorAccount deletes a vendor_account record
func (d *dal) DeleteVendorAccount(ctx context.Context, id VendorAccountID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM vendor_account
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

const selectVendorAccount = `
    SELECT vendor_account.created, vendor_account.id, vendor_account.entity_id, vendor_account.account_type, vendor_account.lifecycle, vendor_account.change_state, vendor_account.modified, vendor_account.access_token, vendor_account.refresh_token, vendor_account.publishable_key, vendor_account.connected_account_id, vendor_account.live, vendor_account.scope
      FROM vendor_account`

func scanVendorAccount(row dbutil.Scanner, contextFormat string, args ...interface{}) (*VendorAccount, error) {
	var m VendorAccount
	m.ID = EmptyVendorAccountID()

	err := row.Scan(&m.Created, &m.ID, &m.EntityID, &m.AccountType, &m.Lifecycle, &m.ChangeState, &m.Modified, &m.AccessToken, &m.RefreshToken, &m.PublishableKey, &m.ConnectedAccountID, &m.Live, &m.Scope)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - vendor_account - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

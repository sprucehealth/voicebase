package dal

import (
	"database/sql"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	InsertAccount(model *Account) (AccountID, error)
	Account(id AccountID) (*Account, error)
	AccountForEmail(email string) (*Account, error)
	UpdateAccount(id AccountID, update *AccountUpdate) (int64, error)
	DeleteAccount(id AccountID) (int64, error)
	InsertAuthToken(model *AuthToken) error
	AuthToken(token string, expiresAfter time.Time) (*AuthToken, error)
	DeleteAuthTokens(accountID AccountID) (int64, error)
	DeleteAuthTokensWithSuffix(accountID AccountID, suffix string) (int64, error)
	DeleteAuthToken(token string) (int64, error)
	UpdateAuthToken(token string, update *AuthTokenUpdate) (int64, error)
	InsertAccountEvent(model *AccountEvent) (AccountEventID, error)
	AccountEvent(id AccountEventID) (*AccountEvent, error)
	DeleteAccountEvent(id AccountEventID) (int64, error)
	InsertAccountPhone(model *AccountPhone) (AccountPhoneID, error)
	AccountPhone(id AccountPhoneID) (*AccountPhone, error)
	UpdateAccountPhone(id AccountPhoneID, update *AccountPhoneUpdate) (int64, error)
	DeleteAccountPhone(id AccountPhoneID) (int64, error)
	InsertAccountEmail(model *AccountEmail) (AccountEmailID, error)
	AccountEmail(id AccountEmailID) (*AccountEmail, error)
	UpdateAccountEmail(id AccountEmailID, update *AccountEmailUpdate) (int64, error)
	DeleteAccountEmail(id AccountEmailID) (int64, error)
	Transact(trans func(dal DAL) error) (err error)
}

type dal struct {
	db tsql.DB
}

// New returns an initialized instance of dal
func New(db *sql.DB) DAL {
	golog.Debugf("Entering dal.New...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.New...") }()
	}
	return &dal{db: tsql.AsDB(db)}
}

// Transact encapsulated the provided function in a transaction and handles rollback and commit actions
func (d *dal) Transact(trans func(dal DAL) error) (err error) {
	golog.Debugf("Entering dal.dal.Transact...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.Transact...") }()
	}
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
			golog.Errorf(string(debug.Stack()))
			err = errors.Trace(fmt.Errorf("Encountered panic during transaction execution: %v", r))
		}
	}()
	if err := trans(tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

// NewAccountPhoneID returns a new AccountPhoneID using the provided value. If id is 0
// then the returned AccountPhoneID is tagged as invalid.
func NewAccountPhoneID(id uint64) AccountPhoneID {
	golog.Debugf("Entering dal.NewAccountPhoneID...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.NewAccountPhoneID...") }()
	}
	return AccountPhoneID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// AccountPhoneID is the ID for a account_phone object
type AccountPhoneID struct {
	encoding.ObjectID
}

// NewAccountEmailID returns a new AccountEmailID using the provided value. If id is 0
// then the returned AccountEmailID is tagged as invalid.
func NewAccountEmailID(id uint64) AccountEmailID {
	golog.Debugf("Entering dal.NewAccountEmailID...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.NewAccountEmailID...") }()
	}
	return AccountEmailID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// AccountEmailID is the ID for a account_email object
type AccountEmailID struct {
	encoding.ObjectID
}

// NewAccountID returns a new AccountID using the provided value. If id is 0
// then the returned AccountID is tagged as invalid.
func NewAccountID(id uint64) AccountID {
	golog.Debugf("Entering dal.NewAccountID...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.NewAccountID...") }()
	}
	return AccountID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// AccountID is the ID for a account object
type AccountID struct {
	encoding.ObjectID
}

const (
	accountIDPrefix = "account"
)

func (a AccountID) String() string {
	golog.Debugf("Entering dal.AccountID.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.AccountID.String...") }()
	}
	return fmt.Sprintf("%s:%d", accountIDPrefix, a.Uint64())
}

// ParseAccountID transforms the provided id string to a numberic value
func ParseAccountID(id string) AccountID {
	golog.Debugf("Entering dal.ParseAccountID...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseAccountID...") }()
	}
	var accountID AccountID
	seg := strings.Split(id, ":")
	if len(seg) > 1 {
		conc.Go(func() {
			if !strings.EqualFold(seg[len(seg)-2], accountIDPrefix) {
				golog.Errorf("%s was provided as an EntityContactID but does not match prefix %s. Continuing anyway.", id, accountIDPrefix)
			}
		})
		id, err := strconv.ParseInt(seg[1], 10, 64)
		if err == nil {
			accountID = NewAccountID(uint64(id))
		} else {
			golog.Warningf("Error while parsing account ID: %s", err)
		}
	}
	return accountID
}

// NewAccountEventID returns a new AccountEventID using the provided value. If id is 0
// then the returned AccountEventID is tagged as invalid.
func NewAccountEventID(id uint64) AccountEventID {
	golog.Debugf("Entering dal.NewAccountEventID...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.NewAccountEventID...") }()
	}
	return AccountEventID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// AccountEventID is the ID for a account_event object
type AccountEventID struct {
	encoding.ObjectID
}

// AccountStatus represents the type associated with the status column of the account table
type AccountStatus string

const (
	// AccountStatusActive represents the ACTIVE state of the status field on a account record
	AccountStatusActive AccountStatus = "ACTIVE"
	// AccountStatusDeleted represents the DELETED state of the status field on a account record
	AccountStatusDeleted AccountStatus = "DELETED"
	// AccountStatusSuspended represents the SUSPENDED state of the status field on a account record
	AccountStatusSuspended AccountStatus = "SUSPENDED"
)

// ParseAccountStatus converts a string into the correcponding enum value
func ParseAccountStatus(s string) (AccountStatus, error) {
	golog.Debugf("Entering dal.ParseAccountStatus...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseAccountStatus...") }()
	}
	switch t := AccountStatus(strings.ToUpper(s)); t {
	case AccountStatusActive, AccountStatusDeleted, AccountStatusSuspended:
		return t, nil
	}
	return AccountStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t AccountStatus) String() string {
	golog.Debugf("Entering dal.AccountStatus.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.AccountStatus.String...") }()
	}
	return string(t)
}

// Scan allows for scanning of AccountStatus from a database conforming to the sql.Scanner interface
func (t *AccountStatus) Scan(src interface{}) error {
	golog.Debugf("Entering dal.AccountStatus.Scan...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.AccountStatus.Scan...") }()
	}
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseAccountStatus(ts)
	case []byte:
		*t, err = ParseAccountStatus(string(ts))
	}
	return errors.Trace(err)
}

// AccountPhoneStatus represents the type associated with the status column of the account_phone table
type AccountPhoneStatus string

const (
	// AccountPhoneStatusActive represents the ACTIVE state of the status field on a account_phone record
	AccountPhoneStatusActive AccountPhoneStatus = "ACTIVE"
	// AccountPhoneStatusDeleted represents the DELETED state of the status field on a account_phone record
	AccountPhoneStatusDeleted AccountPhoneStatus = "DELETED"
	// AccountPhoneStatusSuspended represents the SUSPENDED state of the status field on a account_phone record
	AccountPhoneStatusSuspended AccountPhoneStatus = "SUSPENDED"
)

// ParseAccountPhoneStatus converts a string into the correcponding enum value
func ParseAccountPhoneStatus(s string) (AccountPhoneStatus, error) {
	golog.Debugf("Entering dal.ParseAccountPhoneStatus...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseAccountPhoneStatus...") }()
	}
	switch t := AccountPhoneStatus(strings.ToUpper(s)); t {
	case AccountPhoneStatusActive, AccountPhoneStatusDeleted, AccountPhoneStatusSuspended:
		return t, nil
	}
	return AccountPhoneStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t AccountPhoneStatus) String() string {
	golog.Debugf("Entering dal.AccountPhoneStatus.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.AccountPhoneStatus.String...") }()
	}
	return string(t)
}

// Scan allows for scanning of AccountPhoneStatus from a database conforming to the sql.Scanner interface
func (t *AccountPhoneStatus) Scan(src interface{}) error {
	golog.Debugf("Entering dal.AccountPhoneStatus.Scan...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.AccountPhoneStatus.Scan...") }()
	}
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseAccountPhoneStatus(ts)
	case []byte:
		*t, err = ParseAccountPhoneStatus(string(ts))
	}
	return errors.Trace(err)
}

// AccountEmailStatus represents the type associated with the status column of the account_email table
type AccountEmailStatus string

const (
	// AccountEmailStatusActive represents the ACTIVE state of the status field on a account_email record
	AccountEmailStatusActive AccountEmailStatus = "ACTIVE"
	// AccountEmailStatusDeleted represents the DELETED state of the status field on a account_email record
	AccountEmailStatusDeleted AccountEmailStatus = "DELETED"
	// AccountEmailStatusSuspended represents the SUSPENDED state of the status field on a account_email record
	AccountEmailStatusSuspended AccountEmailStatus = "SUSPENDED"
)

// ParseAccountEmailStatus converts a string into the correcponding enum value
func ParseAccountEmailStatus(s string) (AccountEmailStatus, error) {
	golog.Debugf("Entering dal.ParseAccountEmailStatus...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseAccountEmailStatus...") }()
	}
	switch t := AccountEmailStatus(strings.ToUpper(s)); t {
	case AccountEmailStatusActive, AccountEmailStatusDeleted, AccountEmailStatusSuspended:
		return t, nil
	}
	return AccountEmailStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t AccountEmailStatus) String() string {
	golog.Debugf("Entering dal.AccountEmailStatus.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.AccountEmailStatus.String...") }()
	}
	return string(t)
}

// Scan allows for scanning of AccountEmailStatus from a database conforming to the sql.Scanner interface
func (t *AccountEmailStatus) Scan(src interface{}) error {
	golog.Debugf("Entering dal.AccountEmailStatus.Scan...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.AccountEmailStatus.Scan...") }()
	}
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseAccountEmailStatus(ts)
	case []byte:
		*t, err = ParseAccountEmailStatus(string(ts))
	}
	return errors.Trace(err)
}

// AuthToken represents a auth_token record
type AuthToken struct {
	AccountID AccountID
	Created   time.Time
	Expires   time.Time
	Token     []byte
}

// AuthTokenUpdate represents the mutable aspects of a auth_token record
type AuthTokenUpdate struct {
	Expires *time.Time
}

// Account represents a account record
type Account struct {
	FirstName             string
	LastName              string
	PrimaryAccountEmailID *AccountEmailID
	Modified              time.Time
	ID                    AccountID
	PrimaryAccountPhoneID *AccountPhoneID
	Password              []byte
	Status                AccountStatus
	Created               time.Time
}

// AccountUpdate represents the mutable aspects of a account record
type AccountUpdate struct {
	PrimaryAccountPhoneID *AccountPhoneID
	Password              *[]byte
	Status                *AccountStatus
	FirstName             *string
	LastName              *string
	PrimaryAccountEmailID *AccountEmailID
}

// AccountEmail represents a account_email record
type AccountEmail struct {
	AccountID AccountID
	Email     string
	Status    AccountEmailStatus
	Verified  bool
	Created   time.Time
	Modified  time.Time
	ID        AccountEmailID
}

// AccountEmailUpdate represents the mutable aspects of a account_email record
type AccountEmailUpdate struct {
	Email    *string
	Status   *AccountEmailStatus
	Verified *bool
}

// AccountPhone represents a account_phone record
type AccountPhone struct {
	Status      AccountPhoneStatus
	Verified    bool
	Created     time.Time
	Modified    time.Time
	ID          AccountPhoneID
	AccountID   AccountID
	PhoneNumber string
}

// AccountPhoneUpdate represents the mutable aspects of a account_phone record
type AccountPhoneUpdate struct {
	PhoneNumber *string
	Status      *AccountPhoneStatus
	Verified    *bool
}

// AccountEvent represents a account_event record
type AccountEvent struct {
	AccountPhoneID *AccountPhoneID
	Event          string
	ID             AccountEventID
	AccountID      *AccountID
	AccountEmailID *AccountEmailID
}

func (d *dal) InsertAccount(model *Account) (AccountID, error) {
	golog.Debugf("Entering dal.dal.InsertAccount: %+v", model)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertAccount...") }()
	}
	if !model.ID.IsValid {
		id, err := idgen.NewID()
		if err != nil {
			return NewAccountID(0), errors.Trace(err)
		}
		model.ID = NewAccountID(id)
	}

	var primaryAccountPhoneID *uint64
	var primaryAccountEmailID *uint64
	if model.PrimaryAccountPhoneID != nil {
		primaryAccountPhoneID = ptr.Uint64(model.PrimaryAccountPhoneID.Uint64())
	}
	if model.PrimaryAccountEmailID != nil {
		primaryAccountEmailID = ptr.Uint64(model.PrimaryAccountEmailID.Uint64())
	}

	if _, err := d.db.Exec(
		`INSERT INTO account
          (password, status, id, primary_account_phone_id, primary_account_email_id, first_name, last_name)
          VALUES (?, ?, ?, ?, ?, ?, ?)`, model.Password, model.Status.String(), model.ID.Uint64(), primaryAccountPhoneID, primaryAccountEmailID, model.FirstName, model.LastName); err != nil {
		return NewAccountID(0), errors.Trace(err)
	}

	return NewAccountID(model.ID.Uint64()), nil
}

func (d *dal) Account(id AccountID) (*Account, error) {
	golog.Debugf("Entering dal.dal.Account: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.Account...") }()
	}
	var idv uint64
	var primaryAccountPhoneIDv *uint64
	var primaryAccountEmailIDv *uint64
	model := &Account{}
	if err := d.db.QueryRow(
		`SELECT password, status, created, id, primary_account_phone_id, primary_account_email_id, modified, first_name, last_name
          FROM account
          WHERE id = ?`, id.Uint64()).Scan(&model.Password, &model.Status, &model.Created, &idv, &primaryAccountPhoneIDv, &primaryAccountEmailIDv, &model.Modified, &model.FirstName, &model.LastName); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("account not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.ID = NewAccountID(idv)
	if primaryAccountPhoneIDv != nil {
		nID := NewAccountPhoneID(*primaryAccountPhoneIDv)
		model.PrimaryAccountPhoneID = &nID
	}
	if primaryAccountEmailIDv != nil {
		nID := NewAccountEmailID(*primaryAccountEmailIDv)
		model.PrimaryAccountEmailID = &nID
	}

	return model, nil
}

func (d *dal) AccountForEmail(email string) (*Account, error) {
	golog.Debugf("Entering dal.dal.AccountForEmail: %s", email)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.AccountForEmail...") }()
	}
	var idv uint64
	var primaryAccountPhoneIDv *uint64
	var primaryAccountEmailIDv *uint64
	model := &Account{}
	if err := d.db.QueryRow(
		`SELECT account.password, account.status, account.created, account.id, account.primary_account_phone_id, account.primary_account_email_id, account.modified, account.first_name, account.last_name
          FROM account
		  JOIN account_email ON account.id = account_email.account_id
          WHERE account_email.email = ?`, email).Scan(&model.Password, &model.Status, &model.Created, &idv, &primaryAccountPhoneIDv, &primaryAccountEmailIDv, &model.Modified, &model.FirstName, &model.LastName); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("account not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.ID = NewAccountID(idv)
	if primaryAccountPhoneIDv != nil {
		nID := NewAccountPhoneID(*primaryAccountPhoneIDv)
		model.PrimaryAccountPhoneID = &nID
	}
	if primaryAccountEmailIDv != nil {
		nID := NewAccountEmailID(*primaryAccountEmailIDv)
		model.PrimaryAccountEmailID = &nID
	}

	return model, nil
}

func (d *dal) UpdateAccount(id AccountID, update *AccountUpdate) (int64, error) {
	golog.Debugf("Entering dal.dal.UpdateAccount: ID: %s, Update: %+v", id, update)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.UpdateAccount...") }()
	}
	args := dbutil.MySQLVarArgs()
	if update.Password != nil {
		args.Append("password", *update.Password)
	}
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if update.PrimaryAccountPhoneID != nil {
		args.Append("primary_account_phone_id", *update.PrimaryAccountPhoneID)
	}
	if update.PrimaryAccountEmailID != nil {
		args.Append("primary_account_email_id", *update.PrimaryAccountEmailID)
	}
	if update.FirstName != nil {
		args.Append("first_name", *update.FirstName)
	}
	if update.LastName != nil {
		args.Append("last_name", *update.LastName)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE account
          SET `+args.Columns()+` WHERE id = ?`, append(args.Values(), id.Uint64())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteAccount(id AccountID) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteAccount: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteAccount...") }()
	}
	res, err := d.db.Exec(
		`DELETE FROM account
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) AuthToken(token string, expiresAfter time.Time) (*AuthToken, error) {
	golog.Debugf("Entering dal.dal.AuthToken: Token: %s, ExpiresAfter: %+v", token, expiresAfter)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.AuthToken...") }()
	}
	authToken := &AuthToken{}
	var aID uint64
	if err := d.db.QueryRow(
		`SELECT token, account_id, created, expires 
		  FROM auth_token
          WHERE token = BINARY ? AND expires > ?`, token, expiresAfter).Scan(&authToken.Token, &aID, &authToken.Created, &authToken.Expires); err == sql.ErrNoRows {
		return nil, api.ErrNotFound("auth_token not found")
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	authToken.AccountID = NewAccountID(aID)
	return authToken, nil
}

func (d *dal) InsertAuthToken(model *AuthToken) error {
	golog.Debugf("Entering dal.dal.InsertAuthToken: %+v", model)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertAuthToken...") }()
	}
	_, err := d.db.Exec(
		`INSERT INTO auth_token
          (account_id, expires, token)
          VALUES (?, ?, ?)`, model.AccountID.Uint64(), model.Expires, model.Token)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *dal) DeleteAuthTokens(id AccountID) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteAuthTokens: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteAuthTokens...") }()
	}
	res, err := d.db.Exec(
		`DELETE FROM auth_token
          WHERE account_id = ?`, id.Uint64())
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteAuthTokensWithSuffix(accountID AccountID, suffix string) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteAuthTokensWithSuffix: %s", suffix)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteAuthTokensWithSuffix...") }()
	}
	if suffix == "" {
		return 0, nil
	}
	res, err := d.db.Exec(
		`DELETE FROM auth_token
          WHERE account_id = ? AND token like '%?'`, suffix)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteAuthToken(token string) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteAuthToken: %s", token)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteAuthToken...") }()
	}
	res, err := d.db.Exec(
		`DELETE FROM auth_token
          WHERE token = BINARY ?`, token)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) UpdateAuthToken(token string, update *AuthTokenUpdate) (int64, error) {
	golog.Debugf("Entering dal.dal.UpdateAuthToken: Token: %s, Update: %+v", token, update)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.UpdateAuthToken...") }()
	}
	args := dbutil.MySQLVarArgs()
	if update.Expires != nil {
		args.Append("expires", *update.Expires)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE auth_token
          SET `+args.Columns()+` WHERE token = ?`, append(args.Values(), []byte(token))...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) InsertAccountEvent(model *AccountEvent) (AccountEventID, error) {
	golog.Debugf("Entering dal.dal.InsertAccountEvent: %+v", model)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertAccountEvent...") }()
	}
	if !model.ID.IsValid {
		id, err := idgen.NewID()
		if err != nil {
			return NewAccountEventID(0), errors.Trace(err)
		}
		model.ID = NewAccountEventID(id)
	}

	var accountPhoneID *uint64
	var accountEmailID *uint64
	var accountID *uint64
	if model.AccountPhoneID != nil {
		accountPhoneID = ptr.Uint64(model.AccountPhoneID.Uint64())
	}
	if model.AccountEmailID != nil {
		accountEmailID = ptr.Uint64(model.AccountEmailID.Uint64())
	}
	if model.AccountID != nil {
		accountID = ptr.Uint64(model.AccountID.Uint64())
	}
	if _, err := d.db.Exec(
		`INSERT INTO account_event
          (account_phone_id, event, id, account_id, account_email_id)
          VALUES (?, ?, ?, ?, ?)`, accountPhoneID, model.Event, model.ID.Uint64(), accountID, accountEmailID); err != nil {
		return NewAccountEventID(0), errors.Trace(err)
	}

	return NewAccountEventID(model.ID.Uint64()), nil
}

func (d *dal) AccountEvent(id AccountEventID) (*AccountEvent, error) {
	golog.Debugf("Entering dal.dal.AccountEvent: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.AccountEvent...") }()
	}
	var accountIDv *uint64
	var accountEmailIDv *uint64
	var accountPhoneIDv *uint64
	var idv uint64
	model := &AccountEvent{}
	if err := d.db.QueryRow(
		`SELECT account_id, account_email_id, account_phone_id, event, id
          FROM account_event
          WHERE id = ?`, id.Uint64()).Scan(&accountIDv, &accountEmailIDv, &accountPhoneIDv, &model.Event, &idv); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("account_event not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	if accountIDv != nil {
		nID := NewAccountID(*accountIDv)
		model.AccountID = &nID
	}
	if accountEmailIDv != nil {
		nID := NewAccountEmailID(*accountEmailIDv)
		model.AccountEmailID = &nID
	}
	if accountPhoneIDv != nil {
		nID := NewAccountPhoneID(*accountPhoneIDv)
		model.AccountPhoneID = &nID
	}
	model.ID = NewAccountEventID(idv)

	return model, nil
}

func (d *dal) DeleteAccountEvent(id AccountEventID) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteAccountEvent: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteAccountEvent...") }()
	}
	res, err := d.db.Exec(
		`DELETE FROM account_event
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) InsertAccountPhone(model *AccountPhone) (AccountPhoneID, error) {
	golog.Debugf("Entering dal.dal.InsertAccountPhone: %+v", model)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertAccountPhone...") }()
	}
	if !model.ID.IsValid {
		id, err := idgen.NewID()
		if err != nil {
			return NewAccountPhoneID(0), errors.Trace(err)
		}
		model.ID = NewAccountPhoneID(id)
	}

	if _, err := d.db.Exec(
		`INSERT INTO account_phone
          (phone_number, status, verified, id, account_id)
          VALUES (?, ?, ?, ?, ?)`, model.PhoneNumber, model.Status.String(), model.Verified, model.ID.Uint64(), model.AccountID.Uint64()); err != nil {
		return NewAccountPhoneID(0), errors.Trace(err)
	}

	return NewAccountPhoneID(model.ID.Uint64()), nil
}

func (d *dal) AccountPhone(id AccountPhoneID) (*AccountPhone, error) {
	golog.Debugf("Entering dal.dal.AccountPhone: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.AccountPhone...") }()
	}
	var idv uint64
	var accountIDv uint64
	model := &AccountPhone{}
	if err := d.db.QueryRow(
		`SELECT verified, created, modified, id, account_id, phone_number, status
          FROM account_phone
          WHERE id = ?`, id.Uint64()).Scan(&model.Verified, &model.Created, &model.Modified, &idv, &accountIDv, &model.PhoneNumber, &model.Status); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("account_phone not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.ID = NewAccountPhoneID(idv)
	model.AccountID = NewAccountID(accountIDv)
	return model, nil
}

func (d *dal) UpdateAccountPhone(id AccountPhoneID, update *AccountPhoneUpdate) (int64, error) {
	golog.Debugf("Entering dal.dal.UpdateAccountPhone: ID: %s, Update: %+v", id, update)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.UpdateAccountPhone...") }()
	}
	args := dbutil.MySQLVarArgs()
	if update.PhoneNumber != nil {
		args.Append("phone_number", *update.PhoneNumber)
	}
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if update.Verified != nil {
		args.Append("verified", *update.Verified)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE account_phone
          SET `+args.Columns()+` WHERE id = ?`, append(args.Values(), id.Uint64())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteAccountPhone(id AccountPhoneID) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteAccountPhone: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteAccountPhone...") }()
	}
	res, err := d.db.Exec(
		`DELETE FROM account_phone
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) InsertAccountEmail(model *AccountEmail) (AccountEmailID, error) {
	golog.Debugf("Entering dal.dal.InsertAccountEmail: %+v", model)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertAccountEmail...") }()
	}
	if !model.ID.IsValid {
		id, err := idgen.NewID()
		if err != nil {
			return NewAccountEmailID(0), errors.Trace(err)
		}
		model.ID = NewAccountEmailID(id)
	}

	if _, err := d.db.Exec(
		`INSERT INTO account_email
          (verified, id, account_id, email, status)
          VALUES (?, ?, ?, ?, ?)`, model.Verified, model.ID.Uint64(), model.AccountID.Uint64(), model.Email, model.Status.String()); err != nil {
		return NewAccountEmailID(0), errors.Trace(err)
	}

	return NewAccountEmailID(model.ID.Uint64()), nil
}

func (d *dal) AccountEmail(id AccountEmailID) (*AccountEmail, error) {
	golog.Debugf("Entering dal.dal.AccountEmail: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.AccountEmail...") }()
	}
	var idv uint64
	var accountIDv uint64
	model := &AccountEmail{}
	if err := d.db.QueryRow(
		`SELECT id, account_id, email, status, verified, created, modified
          FROM account_email
          WHERE id = ?`, id.Uint64()).Scan(&idv, &accountIDv, &model.Email, &model.Status, &model.Verified, &model.Created, &model.Modified); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("account_email not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.ID = NewAccountEmailID(idv)
	model.AccountID = NewAccountID(accountIDv)
	return model, nil
}

func (d *dal) UpdateAccountEmail(id AccountEmailID, update *AccountEmailUpdate) (int64, error) {
	golog.Debugf("Entering dal.dal.UpdateAccountEmail: ID: %s, Update: %+v", id, update)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.UpdateAccountEmail...") }()
	}
	args := dbutil.MySQLVarArgs()
	if update.Email != nil {
		args.Append("email", *update.Email)
	}
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if update.Verified != nil {
		args.Append("verified", *update.Verified)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE account_email
          SET `+args.Columns()+` WHERE id = ?`, append(args.Values(), id.Uint64())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteAccountEmail(id AccountEmailID) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteAccountEmail: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteAccountEmail...") }()
	}
	res, err := d.db.Exec(
		`DELETE FROM account_email
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

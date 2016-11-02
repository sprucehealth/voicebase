package dal

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	modellib "github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"github.com/sprucehealth/backend/svc/auth"
)

// ErrNotFound is returned when an item is not found
var ErrNotFound = errors.New("auth/dal: item not found")

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	InsertAccount(ctx context.Context, model *Account) (AccountID, error)
	Account(ctx context.Context, id AccountID) (*Account, error)
	AccountForEmail(ctx context.Context, email string) (*Account, error)
	UpdateAccount(ctx context.Context, id AccountID, update *AccountUpdate) (int64, error)
	DeleteAccount(ctx context.Context, id AccountID) (int64, error)
	InsertAuthToken(ctx context.Context, model *AuthToken) error
	ActiveAuthTokenForAccount(ctx context.Context, accountID AccountID, deviceID string, duration AuthTokenDurationType) (*AuthToken, error)
	AuthToken(ctx context.Context, token string, expiresAfter time.Time, forUpdate bool) (*AuthToken, error)
	DeleteExpiredAuthTokens(ctx context.Context, expiredAgo time.Time) (int64, error)
	DeleteAuthTokens(ctx context.Context, accountID AccountID) (int64, error)
	DeleteAuthToken(ctx context.Context, token string) (int64, error)
	UpdateAuthToken(ctx context.Context, token string, update *AuthTokenUpdate) (int64, error)
	InsertAccountEvent(ctx context.Context, model *AccountEvent) (AccountEventID, error)
	AccountEvent(ctx context.Context, id AccountEventID) (*AccountEvent, error)
	DeleteAccountEvent(ctx context.Context, id AccountEventID) (int64, error)
	InsertAccountPhone(ctx context.Context, model *AccountPhone) (AccountPhoneID, error)
	AccountPhone(ctx context.Context, id AccountPhoneID) (*AccountPhone, error)
	AccountPhoneForAccount(ctx context.Context, id AccountID) (*AccountPhone, error)
	UpdateAccountPhone(ctx context.Context, id AccountPhoneID, update *AccountPhoneUpdate) (int64, error)
	DeleteAccountPhone(ctx context.Context, id AccountPhoneID) (int64, error)
	InsertAccountEmail(ctx context.Context, model *AccountEmail) (AccountEmailID, error)
	AccountEmail(ctx context.Context, id AccountEmailID) (*AccountEmail, error)
	AccountEmailForAccount(ctx context.Context, id AccountID) (*AccountEmail, error)
	UpdateAccountEmail(ctx context.Context, id AccountEmailID, update *AccountEmailUpdate) (int64, error)
	DeleteAccountEmail(ctx context.Context, id AccountEmailID) (int64, error)
	InsertVerificationCode(ctx context.Context, model *VerificationCode) error
	UpdateVerificationCode(ctx context.Context, token string, update *VerificationCodeUpdate) (int64, error)
	VerificationCode(ctx context.Context, token string) (*VerificationCode, error)
	VerificationCodesByValue(ctx context.Context, codeType VerificationCodeType, verifiedValue string) ([]*VerificationCode, error)
	DeleteVerificationCode(ctx context.Context, token string) (int64, error)
	TwoFactorLogin(ctx context.Context, accountID AccountID, deviceID string) (*TwoFactorLogin, error)
	TrackLogin(ctx context.Context, accountID AccountID, platform device.Platform, deviceID string) error
	LastLogin(ctx context.Context, accountID AccountID) (*LoginInfo, error)
	UpsertTwoFactorLogin(ctx context.Context, accountID AccountID, deviceID string, loginTime time.Time) error
	Transact(ctx context.Context, trans func(ctx context.Context, dal DAL) error) (err error)
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
			golog.Errorf(string(debug.Stack()))
			err = errors.Trace(fmt.Errorf("Encountered panic during transaction execution: %v", r))
		}
	}()
	if err := trans(ctx, tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

// AccountIDPrefix represents the string that is attached to the beginning of these identifiers
const AccountIDPrefix = auth.AccountIDPrefix

// NewAccountID returns a new AccountID.
func NewAccountID() (AccountID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return AccountID{}, errors.Trace(err)
	}
	return AccountID{
		modellib.ObjectID{
			Prefix:  AccountIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyAccountID returns an empty initialized ID
func EmptyAccountID() AccountID {
	return AccountID{
		modellib.ObjectID{
			Prefix:  AccountIDPrefix,
			IsValid: false,
		},
	}
}

// ParseAccountID transforms an AccountID from it's string representation into the actual ID value
func ParseAccountID(s string) (AccountID, error) {
	id := EmptyAccountID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// AccountID is the ID for a AccountID object
type AccountID struct {
	modellib.ObjectID
}

// AccountEventIDPrefix represents the string that is attached to the beginning of these identifiers
const AccountEventIDPrefix = "accountEvent_"

// NewAccountEventID returns a new AccountEventID.
func NewAccountEventID() (AccountEventID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return AccountEventID{}, errors.Trace(err)
	}
	return AccountEventID{
		modellib.ObjectID{
			Prefix:  AccountEventIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyAccountEventID returns an empty initialized ID
func EmptyAccountEventID() AccountEventID {
	return AccountEventID{
		modellib.ObjectID{
			Prefix:  AccountEventIDPrefix,
			IsValid: false,
		},
	}
}

// ParseAccountEventID transforms an AccountEventID from it's string representation into the actual ID value
func ParseAccountEventID(s string) (AccountEventID, error) {
	id := EmptyAccountEventID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// AccountEventID is the ID for a AccountEventID object
type AccountEventID struct {
	modellib.ObjectID
}

// AccountPhoneIDPrefix represents the string that is attached to the beginning of these identifiers
const AccountPhoneIDPrefix = "accountPhone_"

// NewAccountPhoneID returns a new AccountPhoneID.
func NewAccountPhoneID() (AccountPhoneID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return AccountPhoneID{}, errors.Trace(err)
	}
	return AccountPhoneID{
		modellib.ObjectID{
			Prefix:  AccountPhoneIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyAccountPhoneID returns an empty initialized ID
func EmptyAccountPhoneID() AccountPhoneID {
	return AccountPhoneID{
		modellib.ObjectID{
			Prefix:  AccountPhoneIDPrefix,
			IsValid: false,
		},
	}
}

// ParseAccountPhoneID transforms an AccountPhoneID from it's string representation into the actual ID value
func ParseAccountPhoneID(s string) (AccountPhoneID, error) {
	id := EmptyAccountPhoneID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// AccountPhoneID is the ID for a AccountPhoneID object
type AccountPhoneID struct {
	modellib.ObjectID
}

// AccountEmailIDPrefix represents the string that is attached to the beginning of these identifiers
const AccountEmailIDPrefix = "accountEmail_"

// NewAccountEmailID returns a new AccountEmailID.
func NewAccountEmailID() (AccountEmailID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return AccountEmailID{}, errors.Trace(err)
	}
	return AccountEmailID{
		modellib.ObjectID{
			Prefix:  AccountEmailIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyAccountEmailID returns an empty initialized ID
func EmptyAccountEmailID() AccountEmailID {
	return AccountEmailID{
		modellib.ObjectID{
			Prefix:  AccountEmailIDPrefix,
			IsValid: false,
		},
	}
}

// ParseAccountEmailID transforms an AccountEmailID from it's string representation into the actual ID value
func ParseAccountEmailID(s string) (AccountEmailID, error) {
	id := EmptyAccountEmailID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// AccountEmailID is the ID for a AccountEmailID object
type AccountEmailID struct {
	modellib.ObjectID
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
	switch t := AccountPhoneStatus(strings.ToUpper(s)); t {
	case AccountPhoneStatusActive, AccountPhoneStatusDeleted, AccountPhoneStatusSuspended:
		return t, nil
	}
	return AccountPhoneStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t AccountPhoneStatus) String() string {
	return string(t)
}

// Scan allows for scanning of AccountPhoneStatus from a database conforming to the sql.Scanner interface
func (t *AccountPhoneStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseAccountPhoneStatus(ts)
	case []byte:
		*t, err = ParseAccountPhoneStatus(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
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
	switch t := AccountEmailStatus(strings.ToUpper(s)); t {
	case AccountEmailStatusActive, AccountEmailStatusDeleted, AccountEmailStatusSuspended:
		return t, nil
	}
	return AccountEmailStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t AccountEmailStatus) String() string {
	return string(t)
}

// Scan allows for scanning of AccountEmailStatus from a database conforming to the sql.Scanner interface
func (t *AccountEmailStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseAccountEmailStatus(ts)
	case []byte:
		*t, err = ParseAccountEmailStatus(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
	}
	return errors.Trace(err)
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
	// AccountStatusBlocked represents a state where an account has been deemed inappropriate for our system
	// with all access via the existing identity into the system blocked.
	AccountStatusBlocked AccountStatus = "BLOCKED"
)

// ParseAccountStatus converts a string into the correcponding enum value
func ParseAccountStatus(s string) (AccountStatus, error) {
	switch t := AccountStatus(strings.ToUpper(s)); t {
	case AccountStatusActive, AccountStatusDeleted, AccountStatusSuspended, AccountStatusBlocked:
		return t, nil
	}
	return AccountStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t AccountStatus) String() string {
	return string(t)
}

// Scan allows for scanning of AccountStatus from a database conforming to the sql.Scanner interface
func (t *AccountStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseAccountStatus(ts)
	case []byte:
		*t, err = ParseAccountStatus(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
	}
	return errors.Trace(err)
}

// AccountType represents the type associated with the status column of the account table
type AccountType string

const (
	// AccountTypePatient represents the PATIENT state of the type field on a account record
	AccountTypePatient AccountType = "PATIENT"
	// AccountTypeProvider represents the PROVIDER state of the type field on a account record
	AccountTypeProvider AccountType = "PROVIDER"
)

// ParseAccountType converts a string into the correcponding enum value
func ParseAccountType(s string) (AccountType, error) {
	switch t := AccountType(strings.ToUpper(s)); t {
	case AccountTypePatient, AccountTypeProvider:
		return t, nil
	}
	return AccountType(""), errors.Trace(fmt.Errorf("Unknown type:%s", s))
}

func (t AccountType) String() string {
	return string(t)
}

// Scan allows for scanning of AccountType from a database conforming to the sql.Scanner interface
func (t *AccountType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseAccountType(ts)
	case []byte:
		*t, err = ParseAccountType(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
	}
	return errors.Trace(err)
}

// VerificationCodeType represents the type associated with the verification_type column of the verification_code table
type VerificationCodeType string

const (
	// VerificationCodeTypePhone represents the PHONE state of the verification_type field on a verification_code record
	VerificationCodeTypePhone VerificationCodeType = "PHONE"
	// VerificationCodeTypeEmail represents the EMAIL state of the verification_type field on a verification_code record
	VerificationCodeTypeEmail VerificationCodeType = "EMAIL"
	// VerificationCodeTypeAccount2fa represents the ACCOUNT_2FA state of the verification_type field on a verification_code record
	VerificationCodeTypeAccount2fa VerificationCodeType = "ACCOUNT_2FA"
	// VerificationCodeTypePasswordReset represents the PASSWORD_RESET state of the verification_type field on a verification_code record
	VerificationCodeTypePasswordReset VerificationCodeType = "PASSWORD_RESET"
)

// ParseVerificationCodeType converts a string into the correcponding enum value
func ParseVerificationCodeType(s string) (VerificationCodeType, error) {
	switch t := VerificationCodeType(strings.ToUpper(s)); t {
	case VerificationCodeTypePhone, VerificationCodeTypeEmail, VerificationCodeTypeAccount2fa, VerificationCodeTypePasswordReset:
		return t, nil
	}
	return VerificationCodeType(""), errors.Trace(fmt.Errorf("Unknown verification_type:%s", s))
}

func (t VerificationCodeType) String() string {
	return string(t)
}

// Scan allows for scanning of VerificationCodeType from a database conforming to the sql.Scanner interface
func (t *VerificationCodeType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseVerificationCodeType(ts)
	case []byte:
		*t, err = ParseVerificationCodeType(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
	}
	return errors.Trace(err)
}

// AuthTokenDurationType represents the duration type of an auth token
type AuthTokenDurationType string

const (
	// AuthTokenDurationTypeShort represents the SHORT state of the duration_type field on a auth_token record
	AuthTokenDurationTypeShort AuthTokenDurationType = "SHORT"
	// AuthTokenDurationTypeMedium represents the MEDIUM state of the type duration_type on a auth_token record
	AuthTokenDurationTypeMedium AuthTokenDurationType = "MEDIUM"
	// AuthTokenDurationTypeLong represents the LONG state of the type duration_type on a auth_token record
	AuthTokenDurationTypeLong AuthTokenDurationType = "LONG"
)

// ParseAuthTokenDurationType converts a string into the correcponding enum value
func ParseAuthTokenDurationType(s string) (AuthTokenDurationType, error) {
	switch t := AuthTokenDurationType(strings.ToUpper(s)); t {
	case AuthTokenDurationTypeShort, AuthTokenDurationTypeMedium, AuthTokenDurationTypeLong:
		return t, nil
	}
	return AuthTokenDurationType(""), errors.Trace(fmt.Errorf("Unknown type:%s", s))
}

func (t AuthTokenDurationType) String() string {
	return string(t)
}

// Scan allows for scanning of AuthTokenDurationType from a database conforming to the sql.Scanner interface
func (t *AuthTokenDurationType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseAuthTokenDurationType(ts)
	case []byte:
		*t, err = ParseAuthTokenDurationType(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
	}
	return errors.Trace(err)
}

// VerificationCode represents a verification_code record
type VerificationCode struct {
	Created          time.Time
	Expires          time.Time
	Token            string
	Code             string
	VerificationType VerificationCodeType
	VerifiedValue    string
	Consumed         bool
}

// VerificationCodeUpdate represents the mutable aspects of a verification_code record
type VerificationCodeUpdate struct {
	Consumed *bool
}

// Account represents a account record
type Account struct {
	FirstName             string
	LastName              string
	Created               time.Time
	ID                    AccountID
	PrimaryAccountEmailID AccountEmailID
	PrimaryAccountPhoneID AccountPhoneID
	Password              []byte
	Status                AccountStatus
	Type                  AccountType
	Modified              time.Time
}

// AccountUpdate represents the mutable aspects of a account record
type AccountUpdate struct {
	PrimaryAccountEmailID AccountEmailID
	PrimaryAccountPhoneID AccountPhoneID
	Password              []byte
	Status                *AccountStatus
	LastName              *string
	FirstName             *string
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
	Event          string
	ID             AccountEventID
	AccountID      AccountID
	AccountEmailID AccountEmailID
	AccountPhoneID AccountPhoneID
}

// AuthToken represents a auth_token record
type AuthToken struct {
	Token               []byte
	ClientEncryptionKey []byte
	AccountID           AccountID
	Created             time.Time
	Expires             time.Time
	// A shadow token is a token that exists solely for the purposes
	//  of supporting in flight calls while the master token is rotating
	Shadow       bool
	DurationType AuthTokenDurationType
	DeviceID     string
	Platform     device.Platform
}

// AuthTokenUpdate represents the mutable aspects of a auth_token record
type AuthTokenUpdate struct {
	Token        []byte
	Expires      *time.Time
	DurationType *AuthTokenDurationType
}

// TwoFactorLogin represents a two_factor_login record
type TwoFactorLogin struct {
	AccountID AccountID
	DeviceID  string
	LastLogin time.Time
}

// LoginInfo represents a login event for an account
type LoginInfo struct {
	AccountID AccountID
	Platform  device.Platform
	DeviceID  string
	Time      time.Time
}

// InsertAccount inserts a account record
func (d *dal) InsertAccount(ctx context.Context, model *Account) (AccountID, error) {
	if !model.ID.IsValid {
		id, err := NewAccountID()
		if err != nil {
			return EmptyAccountID(), errors.Trace(err)
		}
		model.ID = id
	}

	//TODO: mraines: Remove this default after the appropriate code has been deployed
	if model.Type == "" {
		model.Type = AccountTypeProvider
	}

	_, err := d.db.Exec(
		`INSERT INTO account
          (first_name, last_name, id, primary_account_email_id, primary_account_phone_id, password, status, type)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, model.FirstName, model.LastName, model.ID, model.PrimaryAccountEmailID, model.PrimaryAccountPhoneID, model.Password, model.Status.String(), model.Type.String())
	if err != nil {
		return EmptyAccountID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Account retrieves a account record
func (d *dal) Account(ctx context.Context, id AccountID) (*Account, error) {
	row := d.db.QueryRow(
		selectAccount+` WHERE id = ?`, id)
	model, err := scanAccount(row, "id = %s", id)
	return model, errors.Trace(err)
}

// AccountForEmail returns the account record associated with the provided email
func (d *dal) AccountForEmail(ctx context.Context, email string) (*Account, error) {
	row := d.db.QueryRow(
		selectAccount+` JOIN account_email ON account.id = account_email.account_id
          WHERE account_email.email = ?`, email)
	model, err := scanAccount(row, "account_email.email = %s", email)
	return model, errors.Trace(err)
}

// UpdateAccount updates the mutable aspects of a account record
func (d *dal) UpdateAccount(ctx context.Context, id AccountID, update *AccountUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.PrimaryAccountEmailID.IsValid {
		args.Append("primary_account_email_id", update.PrimaryAccountEmailID)
	}
	if update.PrimaryAccountPhoneID.IsValid {
		args.Append("primary_account_phone_id", update.PrimaryAccountPhoneID)
	}
	if len(update.Password) != 0 {
		args.Append("password", update.Password)
	}
	if update.Status != nil {
		args.Append("status", update.Status.String())
	}
	if update.LastName != nil {
		args.Append("last_name", *update.LastName)
	}
	if update.FirstName != nil {
		args.Append("first_name", *update.FirstName)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE account
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteAccount deletes a account record
func (d *dal) DeleteAccount(ctx context.Context, id AccountID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM account
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// AuthToken returns the auth token record that conforms to the provided input
func (d *dal) AuthToken(ctx context.Context, token string, expiresAfter time.Time, forUpdate bool) (*AuthToken, error) {
	var fu string
	if forUpdate {
		fu = "FOR UPDATE"
	}
	row := d.db.QueryRow(
		selectAuthToken+` WHERE token = BINARY ? AND expires > ? `+fu, token, expiresAfter)
	model, err := scanAuthToken(row, "token = %s", token)
	return model, errors.Trace(err)
}

// ActiveAuthTokenForAccount returns the current active non shadow auth token record that conforms to the provided input
func (d *dal) ActiveAuthTokenForAccount(ctx context.Context, accountID AccountID, deviceID string, duration AuthTokenDurationType) (*AuthToken, error) {
	now := time.Now()
	row := d.db.QueryRow(
		selectAuthToken+` WHERE account_id = ? AND shadow = false AND expires > ? AND device_id = ? ORDER BY created DESC LIMIT 1`, accountID, now, deviceID)
	model, err := scanAuthToken(row, "account_id = %s, shadow = false, expires > %s, device_id = %s", accountID, now, deviceID)
	return model, errors.Trace(err)
}

// InsertAuthToken inserts a auth_token record
func (d *dal) InsertAuthToken(ctx context.Context, model *AuthToken) error {
	_, err := d.db.Exec(
		`INSERT INTO auth_token
          (token, client_encryption_key, account_id, expires, shadow, duration_type, device_id, platform)
          VALUES (?, ?, ?, ?, ?, ?, ? ,?)`, model.Token, model.ClientEncryptionKey, model.AccountID, model.Expires, model.Shadow, model.DurationType.String(), model.DeviceID, model.Platform.String())
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// DeleteExpiredAuthTokens deleted the auth tokens associated with the provided account id
func (d *dal) DeleteExpiredAuthTokens(ctx context.Context, expiredBefore time.Time) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM auth_token
          WHERE expires < ?`, expiredBefore)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteAuthTokens deleted the auth tokens associated with the provided account id
func (d *dal) DeleteAuthTokens(ctx context.Context, id AccountID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM auth_token
          WHERE account_id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteAuthToken deleted the provided auth token
func (d *dal) DeleteAuthToken(ctx context.Context, token string) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM auth_token
          WHERE token = BINARY ?`, token)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// UpdateAuthToken updated the mutable aspects of the provided token
func (d *dal) UpdateAuthToken(ctx context.Context, token string, update *AuthTokenUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if len(update.Token) != 0 {
		args.Append("token", update.Token)
	}
	if update.Expires != nil {
		args.Append("expires", *update.Expires)
	}
	if update.DurationType != nil {
		args.Append("duration_type", (*update.DurationType).String())
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE auth_token
          SET `+args.ColumnsForUpdate()+` WHERE token = ?`, append(args.Values(), []byte(token))...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertAccountEvent inserts a account_event record
func (d *dal) InsertAccountEvent(ctx context.Context, model *AccountEvent) (AccountEventID, error) {
	if !model.ID.IsValid {
		id, err := NewAccountEventID()
		if err != nil {
			return EmptyAccountEventID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO account_event
          (id, account_id, account_email_id, account_phone_id, event)
          VALUES (?, ?, ?, ?, ?)`, model.ID, model.AccountID, model.AccountEmailID, model.AccountPhoneID, model.Event)
	if err != nil {
		return EmptyAccountEventID(), errors.Trace(err)
	}

	return model.ID, nil
}

// AccountEvent retrieves a account_event record
func (d *dal) AccountEvent(ctx context.Context, id AccountEventID) (*AccountEvent, error) {
	row := d.db.QueryRow(
		selectAccountEvent+` WHERE id = ?`, id)
	model, err := scanAccountEvent(row, "id = %s", id)
	return model, errors.Trace(err)
}

// DeleteAccountEvent deletes a account_event record
func (d *dal) DeleteAccountEvent(ctx context.Context, id AccountEventID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM account_event
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertAccountPhone inserts a account_phone record
func (d *dal) InsertAccountPhone(ctx context.Context, model *AccountPhone) (AccountPhoneID, error) {
	if !model.ID.IsValid {
		id, err := NewAccountPhoneID()
		if err != nil {
			return EmptyAccountPhoneID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO account_phone
          (id, account_id, phone_number, status, verified)
          VALUES (?, ?, ?, ?, ?)`, model.ID, model.AccountID, model.PhoneNumber, model.Status.String(), model.Verified)
	if err != nil {
		return EmptyAccountPhoneID(), errors.Trace(err)
	}

	return model.ID, nil
}

// AccountPhone retrieves a account_phone record
func (d *dal) AccountPhone(ctx context.Context, id AccountPhoneID) (*AccountPhone, error) {
	row := d.db.QueryRow(
		selectAccountPhone+` WHERE id = ?`, id)
	model, err := scanAccountPhone(row, "id = %s", id)
	return model, errors.Trace(err)
}

// AccountPhoneForAccount retrieves a account_phone record
func (d *dal) AccountPhoneForAccount(ctx context.Context, id AccountID) (*AccountPhone, error) {
	row := d.db.QueryRow(
		selectAccountPhone+` WHERE account_id = ?`, id)
	model, err := scanAccountPhone(row, "account_id = %s", id)
	return model, errors.Trace(err)
}

// UpdateAccountPhone updates the mutable aspects of a account_phone record
func (d *dal) UpdateAccountPhone(ctx context.Context, id AccountPhoneID, update *AccountPhoneUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.PhoneNumber != nil {
		args.Append("phone_number", *update.PhoneNumber)
	}
	if update.Status != nil {
		args.Append("status", update.Status.String())
	}
	if update.Verified != nil {
		args.Append("verified", *update.Verified)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE account_phone
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteAccountPhone deletes a account_phone record
func (d *dal) DeleteAccountPhone(ctx context.Context, id AccountPhoneID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM account_phone
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertAccountEmail inserts a account_email record
func (d *dal) InsertAccountEmail(ctx context.Context, model *AccountEmail) (AccountEmailID, error) {
	if !model.ID.IsValid {
		id, err := NewAccountEmailID()
		if err != nil {
			return EmptyAccountEmailID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO account_email
          (id, account_id, email, status, verified)
          VALUES (?, ?, ?, ?, ?)`, model.ID, model.AccountID, model.Email, model.Status.String(), model.Verified)
	if err != nil {
		return EmptyAccountEmailID(), errors.Trace(err)
	}

	return model.ID, nil
}

// AccountEmail retrieves a account_email record
func (d *dal) AccountEmail(ctx context.Context, id AccountEmailID) (*AccountEmail, error) {
	row := d.db.QueryRow(
		selectAccountEmail+` WHERE id = ?`, id)
	model, err := scanAccountEmail(row, "id = %s", id)
	return model, errors.Trace(err)
}

// AccountEmailForAccount retrieves a account_email record
func (d *dal) AccountEmailForAccount(ctx context.Context, id AccountID) (*AccountEmail, error) {
	row := d.db.QueryRow(
		selectAccountEmail+` WHERE account_id = ?`, id)
	model, err := scanAccountEmail(row, "account_id = %s", id)
	return model, errors.Trace(err)
}

// UpdateAccountEmail updates the mutable aspects of a account_email record
func (d *dal) UpdateAccountEmail(ctx context.Context, id AccountEmailID, update *AccountEmailUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Email != nil {
		args.Append("email", *update.Email)
	}
	if update.Status != nil {
		args.Append("status", update.Status.String())
	}
	if update.Verified != nil {
		args.Append("verified", *update.Verified)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE account_email
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteAccountEmail deletes a account_email record
func (d *dal) DeleteAccountEmail(ctx context.Context, id AccountEmailID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM account_email
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertVerificationCode inserts a verification_code record
func (d *dal) InsertVerificationCode(ctx context.Context, model *VerificationCode) error {
	_, err := d.db.Exec(
		`INSERT INTO verification_code
          (expires, token, code, verification_type, verified_value, consumed)
          VALUES (?, ?, ?, ?, ?, ?)`, model.Expires, model.Token, model.Code, model.VerificationType.String(), model.VerifiedValue, model.Consumed)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// UpdateVerificationCode updates the mutable aspects of a verification_code record
func (d *dal) UpdateVerificationCode(ctx context.Context, token string, update *VerificationCodeUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Consumed != nil {
		args.Append("consumed", *update.Consumed)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE verification_code
          SET `+args.ColumnsForUpdate()+` WHERE token = ?`, append(args.Values(), token)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// VerificationCode retrieves a verification_code record
func (d *dal) VerificationCode(ctx context.Context, token string) (*VerificationCode, error) {
	row := d.db.QueryRow(
		selectVerificationCode+` WHERE token = ?`, token)
	model, err := scanVerificationCode(row, "token = %s", token)
	return model, errors.Trace(err)
}

func (d *dal) VerificationCodesByValue(ctx context.Context, codeType VerificationCodeType, verifiedValue string) ([]*VerificationCode, error) {
	rows, err := d.db.Query(selectVerificationCode+` WHERE verification_type = ? AND verified_value = ?`, codeType.String(), verifiedValue)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var verificationCodes []*VerificationCode
	for rows.Next() {
		verificationCode, err := scanVerificationCode(rows, "verification_type = %s, verified_value = %s", codeType.String(), verifiedValue)
		if err != nil {
			return nil, errors.Trace(err)
		}
		verificationCodes = append(verificationCodes, verificationCode)
	}

	return verificationCodes, errors.Trace(rows.Err())
}

// DeleteVerificationCode deletes a verification_code record
func (d *dal) DeleteVerificationCode(ctx context.Context, token string) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM verification_code
          WHERE token = ?`, token)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// TwoFactorLogin retrieves a verification_code record
func (d *dal) TwoFactorLogin(ctx context.Context, accountID AccountID, deviceID string) (*TwoFactorLogin, error) {
	row := d.db.QueryRow(
		selectTwoFactorLogin+` WHERE account_id = ? AND device_id = ?`, accountID, deviceID)
	model, err := scanTwoFactorLogin(row, "account_id = %s, device_id = %s", accountID, deviceID)
	return model, errors.Trace(err)
}

// UpsertTwoFactorLogin inserts a new two factor login record if one doesn't already exist. If it does then the record is updated.
func (d *dal) UpsertTwoFactorLogin(ctx context.Context, accountID AccountID, deviceID string, loginTime time.Time) error {
	_, err := d.db.Exec(
		`INSERT INTO two_factor_login
          (account_id, device_id, last_login)
          VALUES (?, ?, ?)
		  ON DUPLICATE KEY UPDATE last_login=VALUES(last_login)`, accountID, deviceID, loginTime)
	return errors.Trace(err)
}

func (d *dal) TrackLogin(ctx context.Context, accountID AccountID, platform device.Platform, deviceID string) error {
	_, err := d.db.Exec(`
		REPLACE INTO login_info (account_id, platform, device_id) VALUES (?, ?, ?)`, accountID, platform.String(), deviceID)
	return errors.Trace(err)
}

func (d *dal) LastLogin(ctx context.Context, accountID AccountID) (*LoginInfo, error) {
	loginInfo := LoginInfo{
		AccountID: accountID,
	}
	if err := d.db.QueryRow(`
		SELECT platform, device_id, last_login_timestamp
		FROM login_info
		WHERE account_id = ?
		ORDER BY last_login_timestamp DESC
		LIMIT 1`, accountID).Scan(
		&loginInfo.Platform,
		&loginInfo.DeviceID,
		&loginInfo.Time); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Trace(ErrNotFound)
		}
		return nil, errors.Trace(err)
	}

	return &loginInfo, nil
}

const selectAccount = `
    SELECT account.primary_account_phone_id, account.password, account.status, account.created, account.primary_account_email_id, account.first_name, account.last_name, account.modified, account.id, account.type
      FROM account`

func scanAccount(row dbutil.Scanner, contextFormat string, args ...interface{}) (*Account, error) {
	var m Account
	m.PrimaryAccountPhoneID = EmptyAccountPhoneID()
	m.PrimaryAccountEmailID = EmptyAccountEmailID()
	m.ID = EmptyAccountID()

	err := row.Scan(&m.PrimaryAccountPhoneID, &m.Password, &m.Status, &m.Created, &m.PrimaryAccountEmailID, &m.FirstName, &m.LastName, &m.Modified, &m.ID, &m.Type)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - account - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

const selectAuthToken = `
    SELECT auth_token.token, auth_token.client_encryption_key, auth_token.account_id, auth_token.created, auth_token.expires, auth_token.shadow, auth_token.duration_type, auth_token.device_id, auth_token.platform
      FROM auth_token`

func scanAuthToken(row dbutil.Scanner, contextFormat string, args ...interface{}) (*AuthToken, error) {
	var m AuthToken
	m.AccountID = EmptyAccountID()

	err := row.Scan(&m.Token, &m.ClientEncryptionKey, &m.AccountID, &m.Created, &m.Expires, &m.Shadow, &m.DurationType, &m.DeviceID, &m.Platform)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - auth_token - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

const selectAccountEvent = `
    SELECT account_event.id, account_event.account_id, account_event.account_email_id, account_event.account_phone_id, account_event.event
      FROM account_event`

func scanAccountEvent(row dbutil.Scanner, contextFormat string, args ...interface{}) (*AccountEvent, error) {
	var m AccountEvent
	m.ID = EmptyAccountEventID()
	m.AccountID = EmptyAccountID()
	m.AccountEmailID = EmptyAccountEmailID()
	m.AccountPhoneID = EmptyAccountPhoneID()

	err := row.Scan(&m.ID, &m.AccountID, &m.AccountEmailID, &m.AccountPhoneID, &m.Event)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - account_event - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

const selectAccountPhone = `
    SELECT account_phone.account_id, account_phone.phone_number, account_phone.status, account_phone.verified, account_phone.created, account_phone.modified, account_phone.id
      FROM account_phone`

func scanAccountPhone(row dbutil.Scanner, contextFormat string, args ...interface{}) (*AccountPhone, error) {
	var m AccountPhone
	m.AccountID = EmptyAccountID()
	m.ID = EmptyAccountPhoneID()

	err := row.Scan(&m.AccountID, &m.PhoneNumber, &m.Status, &m.Verified, &m.Created, &m.Modified, &m.ID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - account_phone - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

const selectAccountEmail = `
    SELECT account_email.email, account_email.status, account_email.verified, account_email.created, account_email.modified, account_email.id, account_email.account_id
      FROM account_email`

func scanAccountEmail(row dbutil.Scanner, contextFormat string, args ...interface{}) (*AccountEmail, error) {
	var m AccountEmail
	m.ID = EmptyAccountEmailID()
	m.AccountID = EmptyAccountID()

	err := row.Scan(&m.Email, &m.Status, &m.Verified, &m.Created, &m.Modified, &m.ID, &m.AccountID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - account_email - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

const selectVerificationCode = `
    SELECT verification_code.verified_value, verification_code.consumed, verification_code.created, verification_code.expires, verification_code.token, verification_code.code, verification_code.verification_type
      FROM verification_code`

func scanVerificationCode(row dbutil.Scanner, contextFormat string, args ...interface{}) (*VerificationCode, error) {
	var m VerificationCode

	err := row.Scan(&m.VerifiedValue, &m.Consumed, &m.Created, &m.Expires, &m.Token, &m.Code, &m.VerificationType)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - verification_code - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

const selectTwoFactorLogin = `
    SELECT two_factor_login.account_id, two_factor_login.device_id, two_factor_login.last_login
      FROM two_factor_login`

func scanTwoFactorLogin(row dbutil.Scanner, contextFormat string, args ...interface{}) (*TwoFactorLogin, error) {
	var m TwoFactorLogin
	m.AccountID = EmptyAccountID()

	err := row.Scan(&m.AccountID, &m.DeviceID, &m.LastLogin)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - two_factor_login - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

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

	// Customer
	InsertCustomer(ctx context.Context, model *Customer) (CustomerID, error)
	Customer(ctx context.Context, id CustomerID, opts ...QueryOption) (*Customer, error)
	CustomerForVendor(ctx context.Context, vendorAccountID VendorAccountID, entityID string, opts ...QueryOption) (*Customer, error)
	UpdateCustomer(ctx context.Context, id CustomerID, update *CustomerUpdate) (int64, error)
	DeleteCustomer(ctx context.Context, id CustomerID) (int64, error)

	// Payment
	InsertPayment(ctx context.Context, model *Payment) (PaymentID, error)
	Payment(ctx context.Context, id PaymentID, opts ...QueryOption) (*Payment, error)
	UpdatePayment(ctx context.Context, id PaymentID, update *PaymentUpdate) (int64, error)
	DeletePayment(ctx context.Context, id PaymentID) (int64, error)
	PaymentsInState(ctx context.Context, lifecycle PaymentLifecycle, changeState PaymentChangeState, limit int64, opts ...QueryOption) ([]*Payment, error)

	// Payment Method
	InsertPaymentMethod(ctx context.Context, model *PaymentMethod) (PaymentMethodID, error)
	PaymentMethod(ctx context.Context, id PaymentMethodID, opts ...QueryOption) (*PaymentMethod, error)
	UpdatePaymentMethod(ctx context.Context, id PaymentMethodID, update *PaymentMethodUpdate) (int64, error)
	DeletePaymentMethod(ctx context.Context, id PaymentMethodID) (int64, error)
	EntityPaymentMethods(ctx context.Context, vendorAccountID VendorAccountID, entityID string, opts ...QueryOption) ([]*PaymentMethod, error)
	PaymentMethodsWithFingerprint(ctx context.Context, storageFingerprint string, opts ...QueryOption) ([]*PaymentMethod, error)
	PaymentMethodWithFingerprint(ctx context.Context, customerID CustomerID, storageFingerprint string, opts ...QueryOption) (*PaymentMethod, error)

	// Vendor Account
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

// CustomerIDPrefix represents the string that is attached to the beginning of these identifiers
const CustomerIDPrefix = "customer_"

// NewCustomerID returns a new CustomerID.
func NewCustomerID() (CustomerID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return CustomerID{}, errors.Trace(err)
	}
	return CustomerID{
		modellib.ObjectID{
			Prefix:  CustomerIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyCustomerID returns an empty initialized ID
func EmptyCustomerID() CustomerID {
	return CustomerID{
		modellib.ObjectID{
			Prefix:  CustomerIDPrefix,
			IsValid: false,
		},
	}
}

// ParseCustomerID transforms an CustomerID from it's string representation into the actual ID value
func ParseCustomerID(s string) (CustomerID, error) {
	id := EmptyCustomerID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// CustomerID is the ID for a CustomerID object
type CustomerID struct {
	modellib.ObjectID
}

// PaymentMethodIDPrefix represents the string that is attached to the beginning of these identifiers
const PaymentMethodIDPrefix = "paymentMethod_"

// NewPaymentMethodID returns a new PaymentMethodID.
func NewPaymentMethodID() (PaymentMethodID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return PaymentMethodID{}, errors.Trace(err)
	}
	return PaymentMethodID{
		modellib.ObjectID{
			Prefix:  PaymentMethodIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyPaymentMethodID returns an empty initialized ID
func EmptyPaymentMethodID() PaymentMethodID {
	return PaymentMethodID{
		modellib.ObjectID{
			Prefix:  PaymentMethodIDPrefix,
			IsValid: false,
		},
	}
}

// ParsePaymentMethodID transforms an PaymentMethodID from it's string representation into the actual ID value
func ParsePaymentMethodID(s string) (PaymentMethodID, error) {
	id := EmptyPaymentMethodID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// PaymentMethodID is the ID for a PaymentMethodID object
type PaymentMethodID struct {
	modellib.ObjectID
}

// PaymentIDPrefix represents the string that is attached to the beginning of these identifiers
const PaymentIDPrefix = "payment_"

// NewPaymentID returns a new PaymentID.
func NewPaymentID() (PaymentID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return PaymentID{}, errors.Trace(err)
	}
	return PaymentID{
		modellib.ObjectID{
			Prefix:  PaymentIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyPaymentID returns an empty initialized ID
func EmptyPaymentID() PaymentID {
	return PaymentID{
		modellib.ObjectID{
			Prefix:  PaymentIDPrefix,
			IsValid: false,
		},
	}
}

// ParsePaymentID transforms an PaymentID from it's string representation into the actual ID value
func ParsePaymentID(s string) (PaymentID, error) {
	id := EmptyPaymentID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// PaymentID is the ID for a PaymentID object
type PaymentID struct {
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

// CustomerLifecycle represents the type associated with the lifecycle column of the customer table
type CustomerLifecycle string

const (
	// CustomerLifecycleActive represents the ACTIVE state of the lifecycle field on a customer record
	CustomerLifecycleActive CustomerLifecycle = "ACTIVE"
)

// ParseCustomerLifecycle converts a string into the correcponding enum value
func ParseCustomerLifecycle(s string) (CustomerLifecycle, error) {
	switch t := CustomerLifecycle(strings.ToUpper(s)); t {
	case CustomerLifecycleActive:
		return t, nil
	}
	return CustomerLifecycle(""), errors.Trace(fmt.Errorf("Unknown lifecycle:%s", s))
}

func (t CustomerLifecycle) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t CustomerLifecycle) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of CustomerLifecycle from a database conforming to the sql.Scanner interface
func (t *CustomerLifecycle) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseCustomerLifecycle(ts)
	case []byte:
		*t, err = ParseCustomerLifecycle(string(ts))
	}
	return errors.Trace(err)
}

// CustomerStorageType represents the type associated with the storage_type column of the customer table
type CustomerStorageType string

const (
	// CustomerStorageTypeStripe represents the STRIPE state of the storage_type field on a customer record
	CustomerStorageTypeStripe CustomerStorageType = "STRIPE"
)

// ParseCustomerStorageType converts a string into the correcponding enum value
func ParseCustomerStorageType(s string) (CustomerStorageType, error) {
	switch t := CustomerStorageType(strings.ToUpper(s)); t {
	case CustomerStorageTypeStripe:
		return t, nil
	}
	return CustomerStorageType(""), errors.Trace(fmt.Errorf("Unknown storage_type:%s", s))
}

func (t CustomerStorageType) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t CustomerStorageType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of CustomerStorageType from a database conforming to the sql.Scanner interface
func (t *CustomerStorageType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseCustomerStorageType(ts)
	case []byte:
		*t, err = ParseCustomerStorageType(string(ts))
	}
	return errors.Trace(err)
}

// CustomerChangeState represents the type associated with the change_state column of the customer table
type CustomerChangeState string

const (
	// CustomerChangeStateNone represents the NONE state of the change_state field on a customer record
	CustomerChangeStateNone CustomerChangeState = "NONE"
	// CustomerChangeStatePending represents the PENDING state of the change_state field on a customer record
	CustomerChangeStatePending CustomerChangeState = "PENDING"
)

// ParseCustomerChangeState converts a string into the correcponding enum value
func ParseCustomerChangeState(s string) (CustomerChangeState, error) {
	switch t := CustomerChangeState(strings.ToUpper(s)); t {
	case CustomerChangeStateNone, CustomerChangeStatePending:
		return t, nil
	}
	return CustomerChangeState(""), errors.Trace(fmt.Errorf("Unknown change_state:%s", s))
}

func (t CustomerChangeState) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t CustomerChangeState) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of CustomerChangeState from a database conforming to the sql.Scanner interface
func (t *CustomerChangeState) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseCustomerChangeState(ts)
	case []byte:
		*t, err = ParseCustomerChangeState(string(ts))
	}
	return errors.Trace(err)
}

// PaymentMethodLifecycle represents the type associated with the lifecycle column of the payment_method table
type PaymentMethodLifecycle string

const (
	// PaymentMethodLifecycleActive represents the ACTIVE state of the lifecycle field on a payment_method record
	PaymentMethodLifecycleActive PaymentMethodLifecycle = "ACTIVE"
	// PaymentMethodLifecycleDeleted represents the DELETED state of the lifecycle field on a payment_method record
	PaymentMethodLifecycleDeleted PaymentMethodLifecycle = "DELETED"
)

// ParsePaymentMethodLifecycle converts a string into the correcponding enum value
func ParsePaymentMethodLifecycle(s string) (PaymentMethodLifecycle, error) {
	switch t := PaymentMethodLifecycle(strings.ToUpper(s)); t {
	case PaymentMethodLifecycleActive, PaymentMethodLifecycleDeleted:
		return t, nil
	}
	return PaymentMethodLifecycle(""), errors.Trace(fmt.Errorf("Unknown lifecycle:%s", s))
}

func (t PaymentMethodLifecycle) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t PaymentMethodLifecycle) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of PaymentMethodLifecycle from a database conforming to the sql.Scanner interface
func (t *PaymentMethodLifecycle) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParsePaymentMethodLifecycle(ts)
	case []byte:
		*t, err = ParsePaymentMethodLifecycle(string(ts))
	}
	return errors.Trace(err)
}

// PaymentMethodStorageType represents the type associated with the storage_type column of the payment_method table
type PaymentMethodStorageType string

const (
	// PaymentMethodStorageTypeStripe represents the STRIPE state of the storage_type field on a payment_method record
	PaymentMethodStorageTypeStripe PaymentMethodStorageType = "STRIPE"
)

// ParsePaymentMethodStorageType converts a string into the correcponding enum value
func ParsePaymentMethodStorageType(s string) (PaymentMethodStorageType, error) {
	switch t := PaymentMethodStorageType(strings.ToUpper(s)); t {
	case PaymentMethodStorageTypeStripe:
		return t, nil
	}
	return PaymentMethodStorageType(""), errors.Trace(fmt.Errorf("Unknown storage_type:%s", s))
}

func (t PaymentMethodStorageType) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t PaymentMethodStorageType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of PaymentMethodStorageType from a database conforming to the sql.Scanner interface
func (t *PaymentMethodStorageType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParsePaymentMethodStorageType(ts)
	case []byte:
		*t, err = ParsePaymentMethodStorageType(string(ts))
	}
	return errors.Trace(err)
}

// PaymentMethodChangeState represents the type associated with the change_state column of the payment_method table
type PaymentMethodChangeState string

const (
	// PaymentMethodChangeStateNone represents the NONE state of the change_state field on a payment_method record
	PaymentMethodChangeStateNone PaymentMethodChangeState = "NONE"
	// PaymentMethodChangeStatePending represents the PENDING state of the change_state field on a payment_method record
	PaymentMethodChangeStatePending PaymentMethodChangeState = "PENDING"
)

// ParsePaymentMethodChangeState converts a string into the correcponding enum value
func ParsePaymentMethodChangeState(s string) (PaymentMethodChangeState, error) {
	switch t := PaymentMethodChangeState(strings.ToUpper(s)); t {
	case PaymentMethodChangeStateNone, PaymentMethodChangeStatePending:
		return t, nil
	}
	return PaymentMethodChangeState(""), errors.Trace(fmt.Errorf("Unknown change_state:%s", s))
}

func (t PaymentMethodChangeState) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t PaymentMethodChangeState) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of PaymentMethodChangeState from a database conforming to the sql.Scanner interface
func (t *PaymentMethodChangeState) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParsePaymentMethodChangeState(ts)
	case []byte:
		*t, err = ParsePaymentMethodChangeState(string(ts))
	}
	return errors.Trace(err)
}

// PaymentMethodType represents the type associated with the type column of the payment_method table
type PaymentMethodType string

const (
	// PaymentMethodTypeCard represents the CARD state of the type field on a payment_method record
	PaymentMethodTypeCard PaymentMethodType = "CARD"
)

// ParsePaymentMethodType converts a string into the correcponding enum value
func ParsePaymentMethodType(s string) (PaymentMethodType, error) {
	switch t := PaymentMethodType(strings.ToUpper(s)); t {
	case PaymentMethodTypeCard:
		return t, nil
	}
	return PaymentMethodType(""), errors.Trace(fmt.Errorf("Unknown change_state:%s", s))
}

func (t PaymentMethodType) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t PaymentMethodType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of PaymentMethodType from a database conforming to the sql.Scanner interface
func (t *PaymentMethodType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParsePaymentMethodType(ts)
	case []byte:
		*t, err = ParsePaymentMethodType(string(ts))
	}
	return errors.Trace(err)
}

// PaymentLifecycle represents the type associated with the lifecycle column of the payment table
type PaymentLifecycle string

const (
	// PaymentLifecycleSubmitted represents the SUBMITTED state of the lifecycle field on a payment record
	PaymentLifecycleSubmitted PaymentLifecycle = "SUBMITTED"
	// PaymentLifecycleAccepted represents the ACCEPTED state of the lifecycle field on a payment record
	PaymentLifecycleAccepted PaymentLifecycle = "ACCEPTED"
	// PaymentLifecycleProcessing represents the PROCESSING state of the lifecycle field on a payment record
	PaymentLifecycleProcessing PaymentLifecycle = "PROCESSING"
	// PaymentLifecycleErrorProcessing represents the ERROR_PROCESSING state of the lifecycle field on a payment record
	PaymentLifecycleErrorProcessing PaymentLifecycle = "ERROR_PROCESSING"
	// PaymentLifecycleCompleted represents the COMPLETED state of the lifecycle field on a payment record
	PaymentLifecycleCompleted PaymentLifecycle = "COMPLETED"
)

// ParsePaymentLifecycle converts a string into the correcponding enum value
func ParsePaymentLifecycle(s string) (PaymentLifecycle, error) {
	switch t := PaymentLifecycle(strings.ToUpper(s)); t {
	case PaymentLifecycleSubmitted, PaymentLifecycleAccepted, PaymentLifecycleProcessing, PaymentLifecycleErrorProcessing, PaymentLifecycleCompleted:
		return t, nil
	}
	return PaymentLifecycle(""), errors.Trace(fmt.Errorf("Unknown lifecycle:%s", s))
}

func (t PaymentLifecycle) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t PaymentLifecycle) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of PaymentLifecycle from a database conforming to the sql.Scanner interface
func (t *PaymentLifecycle) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParsePaymentLifecycle(ts)
	case []byte:
		*t, err = ParsePaymentLifecycle(string(ts))
	}
	return errors.Trace(err)
}

// PaymentChangeState represents the type associated with the change_state column of the payment table
type PaymentChangeState string

const (
	// PaymentChangeStateNone represents the NONE state of the change_state field on a payment record
	PaymentChangeStateNone PaymentChangeState = "NONE"
	// PaymentChangeStatePending represents the PENDING state of the change_state field on a payment record
	PaymentChangeStatePending PaymentChangeState = "PENDING"
)

// ParsePaymentChangeState converts a string into the correcponding enum value
func ParsePaymentChangeState(s string) (PaymentChangeState, error) {
	switch t := PaymentChangeState(strings.ToUpper(s)); t {
	case PaymentChangeStateNone, PaymentChangeStatePending:
		return t, nil
	}
	return PaymentChangeState(""), errors.Trace(fmt.Errorf("Unknown change_state:%s", s))
}

func (t PaymentChangeState) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t PaymentChangeState) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of PaymentChangeState from a database conforming to the sql.Scanner interface
func (t *PaymentChangeState) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParsePaymentChangeState(ts)
	case []byte:
		*t, err = ParsePaymentChangeState(string(ts))
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

// PaymentMethod represents a payment_method record
type PaymentMethod struct {
	ID                 PaymentMethodID
	VendorAccountID    VendorAccountID
	StorageType        PaymentMethodStorageType
	ChangeState        PaymentMethodChangeState
	Modified           time.Time
	CustomerID         CustomerID
	EntityID           string
	StorageID          string
	StorageFingerprint string
	Lifecycle          PaymentMethodLifecycle
	Created            time.Time
	Type               PaymentMethodType
	Brand              string
	Last4              string
	ExpMonth           int
	ExpYear            int
	TokenizationMethod string
}

// Validate asserts that the object is well formed
func (m *PaymentMethod) Validate() error {
	if m.Lifecycle == "" {
		return errors.New("Lifecycle cannot be empty")
	}
	if m.ChangeState == "" {
		return errors.New("ChangeState cannot be empty")
	}
	if !m.VendorAccountID.IsValid {
		return errors.New("VendorAccountID must be valid")
	}
	if !m.CustomerID.IsValid {
		return errors.New("CustomerID must be valid")
	}
	if m.EntityID == "" {
		return errors.New("EntityID cannot be empty")
	}
	if m.StorageType == "" {
		return errors.New("StorageType cannot be empty")
	}
	if m.StorageID == "" {
		return errors.New("StorageID cannot be empty")
	}
	if m.StorageFingerprint == "" {
		return errors.New("StorageFingerprint cannot be empty")
	}
	return nil
}

// PaymentMethodUpdate represents the mutable aspects of a payment_method record
type PaymentMethodUpdate struct {
	Lifecycle          PaymentMethodLifecycle
	ChangeState        PaymentMethodChangeState
	StorageID          *string
	StorageFingerprint *string
}

// Validate asserts that the object is well formed
func (m *PaymentMethodUpdate) Validate() error {
	if m.Lifecycle == "" {
		return errors.New("Lifecycle cannot be empty")
	}
	if m.ChangeState == "" {
		return errors.New("ChangeState cannot be empty")
	}
	return nil
}

// Customer represents a customer record
type Customer struct {
	Created         time.Time
	Modified        time.Time
	VendorAccountID VendorAccountID
	EntityID        string
	StorageID       string
	Lifecycle       CustomerLifecycle
	ID              CustomerID
	StorageType     CustomerStorageType
	ChangeState     CustomerChangeState
}

// Validate asserts that the object is well formed
func (m *Customer) Validate() error {
	if m.Lifecycle == "" {
		return errors.New("Lifecycle cannot be empty")
	}
	if m.ChangeState == "" {
		return errors.New("ChangeState cannot be empty")
	}
	if !m.VendorAccountID.IsValid {
		return errors.New("VendorAccountID must be valid")
	}
	if m.EntityID == "" {
		return errors.New("EntityID cannot be empty")
	}
	if m.StorageType == "" {
		return errors.New("StorageType cannot be empty")
	}
	if m.StorageID == "" {
		return errors.New("StorageID cannot be empty")
	}
	return nil
}

// CustomerUpdate represents the mutable aspects of a customer record
type CustomerUpdate struct {
	ChangeState CustomerChangeState
	Lifecycle   CustomerLifecycle
}

// Validate asserts that the object is well formed
func (m *CustomerUpdate) Validate() error {
	if m.Lifecycle == "" {
		return errors.New("Lifecycle cannot be empty")
	}
	if m.ChangeState == "" {
		return errors.New("ChangeState cannot be empty")
	}
	return nil
}

// Payment represents a payment record
type Payment struct {
	Created         time.Time
	Modified        time.Time
	ID              PaymentID
	Currency        string
	Amount          uint64
	ChangeState     PaymentChangeState
	VendorAccountID VendorAccountID
	PaymentMethodID PaymentMethodID
	Lifecycle       PaymentLifecycle
}

// Validate asserts that the object is well formed
func (m *Payment) Validate() error {
	if m.Lifecycle == "" {
		return errors.New("Lifecycle cannot be empty")
	}
	if m.ChangeState == "" {
		return errors.New("ChangeState cannot be empty")
	}
	if m.Currency == "" {
		return errors.New("Currency cannot be empty")
	}
	if !m.VendorAccountID.IsValid {
		return errors.New("VendorAccountID must be valid")
	}
	if m.Amount <= 0 {
		return errors.New("Amount must be positive non zero value")
	}
	return nil
}

// PaymentUpdate represents the mutable aspects of a payment record
type PaymentUpdate struct {
	Lifecycle       PaymentLifecycle
	ChangeState     PaymentChangeState
	PaymentMethodID *PaymentMethodID
}

// Validate asserts that the object is well formed
func (m *PaymentUpdate) Validate() error {
	if m.Lifecycle == "" {
		return errors.New("Lifecycle cannot be empty")
	}
	if m.ChangeState == "" {
		return errors.New("ChangeState cannot be empty")
	}
	if m.PaymentMethodID != nil && !m.PaymentMethodID.IsValid {
		return errors.New("PaymentMethodID must be valid")
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
	row := d.db.QueryRow(q, id)
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

// InsertCustomer inserts a customer record
func (d *dal) InsertCustomer(ctx context.Context, model *Customer) (CustomerID, error) {
	if !model.ID.IsValid {
		id, err := NewCustomerID()
		if err != nil {
			return EmptyCustomerID(), errors.Trace(err)
		}
		model.ID = id
	}
	if err := model.Validate(); err != nil {
		return EmptyCustomerID(), errors.Trace(err)
	}
	_, err := d.db.Exec(
		`INSERT INTO customer
          (vendor_account_id, entity_id, storage_id, lifecycle, id, storage_type, change_state)
          VALUES (?, ?, ?, ?, ?, ?, ?)`, model.VendorAccountID, model.EntityID, model.StorageID, model.Lifecycle, model.ID, model.StorageType, model.ChangeState)
	if err != nil {
		return EmptyCustomerID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Customer retrieves a customer record
func (d *dal) Customer(ctx context.Context, id CustomerID, opts ...QueryOption) (*Customer, error) {
	q := selectCustomer + ` WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, id)
	model, err := scanCustomer(row, id.String())
	return model, errors.Trace(err)
}

// CustomerForVendor retrieves a customer record mapping to the provided vendor and entity ids
func (d *dal) CustomerForVendor(ctx context.Context, vendorAccountID VendorAccountID, entityID string, opts ...QueryOption) (*Customer, error) {
	q := selectCustomer + ` WHERE vendor_account_id = ? AND entity_id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, vendorAccountID, entityID)
	model, err := scanCustomer(row, "vendor_account_id: %s - entity_id: %s", vendorAccountID, entityID)
	return model, errors.Trace(err)
}

// UpdateCustomer updates the mutable aspects of a customer record
func (d *dal) UpdateCustomer(ctx context.Context, id CustomerID, update *CustomerUpdate) (int64, error) {
	if update == nil {
		return 0, nil
	}
	if err := update.Validate(); err != nil {
		return 0, errors.Trace(err)
	}

	args := dbutil.MySQLVarArgs()
	args.Append("lifecycle", update.Lifecycle)
	args.Append("change_state", update.ChangeState)
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE customer
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteCustomer deletes a customer record
func (d *dal) DeleteCustomer(ctx context.Context, id CustomerID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM customer
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertPaymentMethod inserts a payment_method record
func (d *dal) InsertPaymentMethod(ctx context.Context, model *PaymentMethod) (PaymentMethodID, error) {
	if !model.ID.IsValid {
		id, err := NewPaymentMethodID()
		if err != nil {
			return EmptyPaymentMethodID(), errors.Trace(err)
		}
		model.ID = id
	}
	if err := model.Validate(); err != nil {
		return EmptyPaymentMethodID(), errors.Trace(err)
	}
	_, err := d.db.Exec(
		`INSERT INTO payment_method
          (id, vendor_account_id, storage_type, storage_fingerprint, change_state, customer_id, entity_id, storage_id, lifecycle, type, brand, last_four, exp_month, exp_year, tokenization_method)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, model.ID, model.VendorAccountID, model.StorageType, model.StorageFingerprint, model.ChangeState, model.CustomerID, model.EntityID, model.StorageID, model.Lifecycle, model.Type, model.Brand, model.Last4, model.ExpMonth, model.ExpYear, model.TokenizationMethod)
	if err != nil {
		return EmptyPaymentMethodID(), errors.Trace(err)
	}

	return model.ID, nil
}

// PaymentMethod retrieves a payment_method record
func (d *dal) PaymentMethod(ctx context.Context, id PaymentMethodID, opts ...QueryOption) (*PaymentMethod, error) {
	q := selectPaymentMethod + ` WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, id)
	model, err := scanPaymentMethod(row, id.String())
	return model, errors.Trace(err)
}

// PaymentMethodWithFingerprint retrieves a payment_method record with the corresponding fingerprint belonging to the specified vendor
func (d *dal) PaymentMethodWithFingerprint(ctx context.Context, customerID CustomerID, storageFingerprint string, opts ...QueryOption) (*PaymentMethod, error) {
	q := selectPaymentMethod + ` WHERE customer_id = ? AND storage_fingerprint = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, customerID, storageFingerprint)
	model, err := scanPaymentMethod(row, "customer_id: %s - storage_fingerprint: %s", customerID, storageFingerprint)
	return model, errors.Trace(err)
}

// PaymentMethodsWithFingerprint retrieves a payment_method records with the corresponding fingerprint
func (d *dal) PaymentMethodsWithFingerprint(ctx context.Context, storageFingerprint string, opts ...QueryOption) ([]*PaymentMethod, error) {
	q := selectPaymentMethod + ` WHERE storage_fingerprint = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, storageFingerprint)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var paymentMethods []*PaymentMethod
	for rows.Next() {
		pm, err := scanPaymentMethod(rows, "storage_fingerprint: %s", storageFingerprint)
		if err != nil {
			return nil, errors.Trace(err)
		}
		paymentMethods = append(paymentMethods, pm)
	}
	return paymentMethods, errors.Trace(rows.Err())
}

// EntityVendorAccounts retrieves a set of payment_method records for the provided entity id
func (d *dal) EntityPaymentMethods(ctx context.Context, vendorAccountID VendorAccountID, entityID string, opts ...QueryOption) ([]*PaymentMethod, error) {
	q := selectPaymentMethod + ` WHERE vendor_account_id = ? AND entity_id = ? AND lifecycle = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	q += ` ORDER BY created DESC`
	rows, err := d.db.Query(q, vendorAccountID, entityID, PaymentMethodLifecycleActive)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var paymentMethods []*PaymentMethod
	for rows.Next() {
		pm, err := scanPaymentMethod(rows, "vendor_account_id: %s - entity_id: %s", vendorAccountID, entityID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		paymentMethods = append(paymentMethods, pm)
	}
	return paymentMethods, errors.Trace(rows.Err())
}

// UpdatePaymentMethod updates the mutable aspects of a payment_method record
func (d *dal) UpdatePaymentMethod(ctx context.Context, id PaymentMethodID, update *PaymentMethodUpdate) (int64, error) {
	if update == nil {
		return 0, nil
	}
	if err := update.Validate(); err != nil {
		return 0, errors.Trace(err)
	}

	args := dbutil.MySQLVarArgs()
	args.Append("lifecycle", update.Lifecycle)
	args.Append("change_state", update.ChangeState)
	if update.StorageID != nil {
		args.Append("storage_id", *update.StorageID)
	}
	if update.StorageFingerprint != nil {
		args.Append("storage_fingerprint", *update.StorageFingerprint)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE payment_method
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeletePaymentMethod deletes a payment_method record
func (d *dal) DeletePaymentMethod(ctx context.Context, id PaymentMethodID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM payment_method
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertPayment inserts a payment record
func (d *dal) InsertPayment(ctx context.Context, model *Payment) (PaymentID, error) {
	if !model.ID.IsValid {
		id, err := NewPaymentID()
		if err != nil {
			return EmptyPaymentID(), errors.Trace(err)
		}
		model.ID = id
	}
	if err := model.Validate(); err != nil {
		return EmptyPaymentID(), errors.Trace(err)
	}
	_, err := d.db.Exec(
		`INSERT INTO payment
          (vendor_account_id, currency, amount, change_state, id, payment_method_id, lifecycle)
          VALUES (?, ?, ?, ?, ?, ?, ?)`, model.VendorAccountID, model.Currency, model.Amount, model.ChangeState, model.ID, model.PaymentMethodID, model.Lifecycle)
	if err != nil {
		return EmptyPaymentID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Payment retrieves a payment record
func (d *dal) Payment(ctx context.Context, id PaymentID, opts ...QueryOption) (*Payment, error) {
	q := selectPayment + ` WHERE id = ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	row := d.db.QueryRow(q, id)
	model, err := scanPayment(row, id.String())
	return model, errors.Trace(err)
}

// PaymentsInState retrieves a random set of payment records with the corresponding state
func (d *dal) PaymentsInState(ctx context.Context, lifecycle PaymentLifecycle, changeState PaymentChangeState, limit int64, opts ...QueryOption) ([]*Payment, error) {
	q := selectPayment + ` WHERE lifecycle = ? AND change_state = ?`
	q += ` ORDER BY RAND() LIMIT ?`
	if queryOptions(opts).Has(ForUpdate) {
		q += ` FOR UPDATE`
	}
	rows, err := d.db.Query(q, lifecycle, changeState, limit)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var payments []*Payment
	for rows.Next() {
		p, err := scanPayment(rows, fmt.Sprintf("lifecycle: %s - change_state: %s - limit: %d", lifecycle, changeState, limit))
		if err != nil {
			return nil, errors.Trace(err)
		}
		payments = append(payments, p)
	}
	return payments, errors.Trace(rows.Err())
}

// UpdatePayment updates the mutable aspects of a payment record
func (d *dal) UpdatePayment(ctx context.Context, id PaymentID, update *PaymentUpdate) (int64, error) {
	if err := update.Validate(); err != nil {
		return 0, errors.Trace(err)
	}

	args := dbutil.MySQLVarArgs()
	args.Append("lifecycle", update.Lifecycle)
	args.Append("change_state", update.ChangeState)
	if update.PaymentMethodID != nil {
		args.Append("payment_method_id", update.PaymentMethodID)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE payment
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeletePayment deletes a payment record
func (d *dal) DeletePayment(ctx context.Context, id PaymentID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM payment
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

const selectCustomer = `
    SELECT customer.id, customer.storage_type, customer.change_state, customer.lifecycle, customer.created, customer.modified, customer.vendor_account_id, customer.entity_id, customer.storage_id
      FROM customer`

func scanCustomer(row dbutil.Scanner, contextFormat string, args ...interface{}) (*Customer, error) {
	var m Customer
	m.ID = EmptyCustomerID()
	m.VendorAccountID = EmptyVendorAccountID()

	err := row.Scan(&m.ID, &m.StorageType, &m.ChangeState, &m.Lifecycle, &m.Created, &m.Modified, &m.VendorAccountID, &m.EntityID, &m.StorageID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - customer - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

const selectPaymentMethod = `
    SELECT payment_method.lifecycle, payment_method.created, payment_method.modified, payment_method.customer_id, payment_method.entity_id, payment_method.storage_id, payment_method.storage_fingerprint, payment_method.change_state, payment_method.id, payment_method.vendor_account_id, payment_method.storage_type, payment_method.type, payment_method.brand, payment_method.last_four, payment_method.exp_month, payment_method.exp_year, payment_method.tokenization_method
      FROM payment_method`

func scanPaymentMethod(row dbutil.Scanner, contextFormat string, args ...interface{}) (*PaymentMethod, error) {
	var m PaymentMethod
	m.CustomerID = EmptyCustomerID()
	m.ID = EmptyPaymentMethodID()
	m.VendorAccountID = EmptyVendorAccountID()

	err := row.Scan(&m.Lifecycle, &m.Created, &m.Modified, &m.CustomerID, &m.EntityID, &m.StorageID, &m.StorageFingerprint, &m.ChangeState, &m.ID, &m.VendorAccountID, &m.StorageType, &m.Type, &m.Brand, &m.Last4, &m.ExpMonth, &m.ExpYear, &m.TokenizationMethod)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - payment_method - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

const selectPayment = `
    SELECT payment.id, payment.payment_method_id, payment.lifecycle, payment.vendor_account_id, payment.currency, payment.amount, payment.change_state, payment.created, payment.modified
      FROM payment`

func scanPayment(row dbutil.Scanner, contextFormat string, args ...interface{}) (*Payment, error) {
	var m Payment
	m.ID = EmptyPaymentID()
	m.PaymentMethodID = EmptyPaymentMethodID()
	m.VendorAccountID = EmptyVendorAccountID()

	err := row.Scan(&m.ID, &m.PaymentMethodID, &m.Lifecycle, &m.VendorAccountID, &m.Currency, &m.Amount, &m.ChangeState, &m.Created, &m.Modified)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errors.Annotate(ErrNotFound, "No rows found - payment - Context: "+fmt.Sprintf(contextFormat, args...)))
	}
	return &m, errors.Trace(err)
}

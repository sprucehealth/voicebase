package testutil

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ dal.DAL = &MockDAL{}

type MockDAL struct {
	*mock.Expector
}

// NewMockDAL returns an initialized instance of MockDAL
func NewMockDAL(t *testing.T) *MockDAL {
	return &MockDAL{&mock.Expector{T: t}}
}

func (d *MockDAL) Transact(ctx context.Context, trans func(ctx context.Context, dl dal.DAL) error) (err error) {
	if err := trans(ctx, d); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (d *MockDAL) DeleteVendorAccount(ctx context.Context, id dal.VendorAccountID) (int64, error) {
	rets := d.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (d *MockDAL) EntityVendorAccounts(ctx context.Context, entityID string, opts ...dal.QueryOption) ([]*dal.VendorAccount, error) {
	var rets []interface{}
	if len(opts) != 0 {
		rets = d.Record([]interface{}{entityID, opts}...)
	} else {
		rets = d.Record(entityID)
	}
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.VendorAccount), mock.SafeError(rets[1])
}

func (d *MockDAL) InsertVendorAccount(ctx context.Context, model *dal.VendorAccount) (dal.VendorAccountID, error) {
	rets := d.Record(model)
	if len(rets) == 0 {
		return dal.EmptyVendorAccountID(), nil
	}
	return rets[0].(dal.VendorAccountID), mock.SafeError(rets[1])
}

func (d *MockDAL) UpdateVendorAccount(ctx context.Context, id dal.VendorAccountID, update *dal.VendorAccountUpdate) error {
	rets := d.Record(id, update)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (d *MockDAL) VendorAccount(ctx context.Context, id dal.VendorAccountID, opts ...dal.QueryOption) (*dal.VendorAccount, error) {
	var rets []interface{}
	if len(opts) != 0 {
		rets = d.Record([]interface{}{id, opts}...)
	} else {
		rets = d.Record(id)
	}
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.VendorAccount), mock.SafeError(rets[1])
}

func (d *MockDAL) VendorAccountsInState(ctx context.Context, lifecycle dal.VendorAccountLifecycle, changeState dal.VendorAccountChangeState, limit int64, opts ...dal.QueryOption) ([]*dal.VendorAccount, error) {
	rets := d.Record(lifecycle, changeState, limit, opts)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.VendorAccount), mock.SafeError(rets[1])
}

func (d *MockDAL) InsertCustomer(ctx context.Context, model *dal.Customer) (dal.CustomerID, error) {
	rets := d.Record(model)
	if len(rets) == 0 {
		return dal.EmptyCustomerID(), nil
	}
	return rets[0].(dal.CustomerID), mock.SafeError(rets[1])
}

func (d *MockDAL) Customer(ctx context.Context, id dal.CustomerID, opts ...dal.QueryOption) (*dal.Customer, error) {
	rets := d.Record(id, opts)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Customer), mock.SafeError(rets[1])
}

func (d *MockDAL) UpdateCustomer(ctx context.Context, id dal.CustomerID, update *dal.CustomerUpdate) (int64, error) {
	rets := d.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (d *MockDAL) DeleteCustomer(ctx context.Context, id dal.CustomerID) (int64, error) {
	rets := d.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (d *MockDAL) CustomerForVendor(ctx context.Context, vendorAccountID dal.VendorAccountID, entityID string, opts ...dal.QueryOption) (*dal.Customer, error) {
	rets := d.Record(vendorAccountID, entityID, opts)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Customer), mock.SafeError(rets[1])
}

func (d *MockDAL) InsertPaymentMethod(ctx context.Context, model *dal.PaymentMethod) (dal.PaymentMethodID, error) {
	rets := d.Record(model)
	if len(rets) == 0 {
		return dal.EmptyPaymentMethodID(), nil
	}
	return rets[0].(dal.PaymentMethodID), mock.SafeError(rets[1])
}

func (d *MockDAL) PaymentMethod(ctx context.Context, id dal.PaymentMethodID, opts ...dal.QueryOption) (*dal.PaymentMethod, error) {
	rets := d.Record(id, opts)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.PaymentMethod), mock.SafeError(rets[1])
}

func (d *MockDAL) UpdatePaymentMethod(ctx context.Context, id dal.PaymentMethodID, update *dal.PaymentMethodUpdate) (int64, error) {
	rets := d.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (d *MockDAL) DeletePaymentMethod(ctx context.Context, id dal.PaymentMethodID) (int64, error) {
	rets := d.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (d *MockDAL) EntityPaymentMethods(ctx context.Context, vendorAccountID dal.VendorAccountID, entityID string, opts ...dal.QueryOption) ([]*dal.PaymentMethod, error) {
	rets := d.Record(vendorAccountID, entityID, opts)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.PaymentMethod), mock.SafeError(rets[1])
}

func (d *MockDAL) PaymentMethodWithFingerprint(ctx context.Context, customerID dal.CustomerID, storageFingerprint string, opts ...dal.QueryOption) (*dal.PaymentMethod, error) {
	rets := d.Record(customerID, storageFingerprint, opts)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.PaymentMethod), mock.SafeError(rets[1])
}

func (d *MockDAL) PaymentMethodsWithFingerprint(ctx context.Context, storageFingerprint string, opts ...dal.QueryOption) ([]*dal.PaymentMethod, error) {
	rets := d.Record(storageFingerprint, opts)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.PaymentMethod), mock.SafeError(rets[1])
}

func (d *MockDAL) InsertPayment(ctx context.Context, model *dal.Payment) (dal.PaymentID, error) {
	rets := d.Record(model)
	if len(rets) == 0 {
		return dal.EmptyPaymentID(), nil
	}
	return rets[0].(dal.PaymentID), mock.SafeError(rets[1])
}

func (d *MockDAL) Payment(ctx context.Context, id dal.PaymentID, opts ...dal.QueryOption) (*dal.Payment, error) {
	rets := d.Record(id, opts)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Payment), mock.SafeError(rets[1])
}

func (d *MockDAL) UpdatePayment(ctx context.Context, id dal.PaymentID, update *dal.PaymentUpdate) (int64, error) {
	rets := d.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (d *MockDAL) DeletePayment(ctx context.Context, id dal.PaymentID) (int64, error) {
	rets := d.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

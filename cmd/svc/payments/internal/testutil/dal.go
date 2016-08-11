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

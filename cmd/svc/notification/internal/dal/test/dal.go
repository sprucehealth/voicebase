package test

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/notification/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

type mockDAL struct{ *mock.Expector }

var _ dal.DAL = NewMockDAL(nil)

// NewMockDAL returns an initialized instance of mockDAL
func NewMockDAL(t *testing.T) *mockDAL {
	return &mockDAL{
		&mock.Expector{T: t},
	}
}

func (d *mockDAL) Transact(trans func(dal.DAL) error) (err error) {
	if err := trans(d); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (d *mockDAL) InsertPushConfig(model *dal.PushConfig) (dal.PushConfigID, error) {
	rets := d.Record(model)
	if len(rets) == 0 {
		return dal.EmptyPushConfigID(), nil
	}
	return rets[0].(dal.PushConfigID), mock.SafeError(rets[1])
}

func (d *mockDAL) PushConfig(id dal.PushConfigID) (*dal.PushConfig, error) {
	rets := d.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.PushConfig), mock.SafeError(rets[1])
}

func (d *mockDAL) PushConfigForDeviceID(deviceID string) (*dal.PushConfig, error) {
	rets := d.Record(deviceID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.PushConfig), mock.SafeError(rets[1])
}

func (d *mockDAL) PushConfigForDeviceToken(deviceToken string) (*dal.PushConfig, error) {
	rets := d.Record(deviceToken)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.PushConfig), mock.SafeError(rets[1])
}

func (d *mockDAL) PushConfigsForExternalGroupID(externalGroupID string) ([]*dal.PushConfig, error) {
	rets := d.Record(externalGroupID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dal.PushConfig), mock.SafeError(rets[1])
}

func (d *mockDAL) UpdatePushConfig(id dal.PushConfigID, update *dal.PushConfigUpdate) (int64, error) {
	rets := d.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (d *mockDAL) DeletePushConfig(id dal.PushConfigID) (int64, error) {
	rets := d.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (d *mockDAL) DeletePushConfigForDeviceID(deviceID string) (int64, error) {
	rets := d.Record(deviceID)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

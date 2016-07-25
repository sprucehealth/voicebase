package mock

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

type mockDAL struct {
	*mock.Expector
}

func New(t *testing.T) *mockDAL {
	return &mockDAL{
		Expector: &mock.Expector{
			T: t,
		},
	}
}

func (d *mockDAL) GetConfigs(keys []string) ([]*models.Config, error) {
	rets := d.Record(keys)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*models.Config), mock.SafeError(rets[1])
}

func (d *mockDAL) SetConfigs(configs []*models.Config) error {
	rets := d.Record(configs)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (d *mockDAL) GetValues(nodeID string, keys []*models.ConfigKey) ([]*models.Value, error) {
	rets := d.Record(nodeID, keys)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*models.Value), mock.SafeError(rets[1])
}

func (d *mockDAL) SetValues(nodeID string, values []*models.Value) error {
	rets := d.Record(nodeID, values)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (d *mockDAL) GetAllConfigs() ([]*models.Config, error) {
	rets := d.Record()
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*models.Config), mock.SafeError(rets[1])
}

func (d *mockDAL) GetNodeValues(nodeID string) ([]*models.Value, error) {
	rets := d.Record(nodeID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*models.Value), mock.SafeError(rets[1])
}

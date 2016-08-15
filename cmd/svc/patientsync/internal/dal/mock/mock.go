package mock

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ dal.DAL = &mockDAL{}

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

func (m *mockDAL) CreateSyncConfig(cfg *sync.Config) error {
	rets := m.Record(cfg)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])

}

func (m *mockDAL) SyncConfigForOrg(orgID string) (*sync.Config, error) {
	rets := m.Record(orgID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*sync.Config), mock.SafeError(rets[1])
}

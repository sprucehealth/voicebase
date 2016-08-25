package mock

import (
	"testing"
	"time"

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

func (m *mockDAL) CreateSyncConfig(cfg *sync.Config, externalID *string) error {
	rets := m.Record(cfg, externalID)
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

func (m *mockDAL) SyncConfigForExternalID(externalID string) (*sync.Config, error) {
	rets := m.Record(externalID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*sync.Config), mock.SafeError(rets[1])
}

func (m *mockDAL) UpdateSyncBookmarkForOrg(orgID string, bookmark time.Time, status dal.SyncStatus) error {
	rets := m.Record(orgID, bookmark, status)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *mockDAL) SyncBookmarkForOrg(orgID string) (*dal.SyncBookmark, error) {
	rets := m.Record(orgID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*dal.SyncBookmark), mock.SafeError(rets[1])
}

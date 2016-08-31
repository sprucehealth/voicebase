package raccess

import (
	"context"

	"github.com/sprucehealth/backend/svc/patientsync"
)

func (m *resourceAccessor) ConfigurePatientSync(ctx context.Context, req *patientsync.ConfigureSyncRequest) (*patientsync.ConfigureSyncResponse, error) {
	if err := m.canAccessResource(ctx, req.OrganizationEntityID, m.orgsForEntity); err != nil {
		return nil, err
	}

	return m.patientsync.ConfigureSync(ctx, req)
}

func (m *resourceAccessor) LookupPatientSyncConfiguration(ctx context.Context, req *patientsync.LookupSyncConfigurationRequest) (*patientsync.LookupSyncConfigurationResponse, error) {
	if err := m.canAccessResource(ctx, req.OrganizationEntityID, m.orgsForEntity); err != nil {
		return nil, err
	}

	return m.patientsync.LookupSyncConfiguration(ctx, req)
}

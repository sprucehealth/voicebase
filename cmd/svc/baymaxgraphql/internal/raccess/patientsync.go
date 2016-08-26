package raccess

import (
	"context"

	"github.com/sprucehealth/backend/svc/patientsync"
)

func (m *resourceAccessor) ConfigurePatientSync(ctx context.Context, req *patientsync.ConfigureSyncRequest) (*patientsync.ConfigureSyncResponse, error) {
	if err := m.canAccessResource(ctx, req.OrganizationEntityID, m.orgsForEntity); err != nil {
		return nil, err
	}

	resp, err := m.patientsync.ConfigureSync(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

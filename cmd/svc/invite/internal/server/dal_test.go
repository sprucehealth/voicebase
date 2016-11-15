package server

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

// Build time check for matching against the interface
var _ dal.DAL = &mockDAL{}

type mockDAL struct {
	*mock.Expector
}

func newMockDAL(t *testing.T) *mockDAL {
	return &mockDAL{&mock.Expector{T: t}}
}

func (dl *mockDAL) AttributionData(ctx context.Context, deviceID string) (map[string]string, error) {
	r := dl.Expector.Record(deviceID)
	return r[0].(map[string]string), mock.SafeError(r[1])
}

func (dl *mockDAL) SetAttributionData(ctx context.Context, deviceID string, values map[string]string) error {
	r := dl.Expector.Record(deviceID, values)
	return mock.SafeError(r[0])
}

func (dl *mockDAL) InsertInvite(ctx context.Context, invite *models.Invite) error {
	r := dl.Expector.Record(invite)
	return mock.SafeError(r[0])
}

func (dl *mockDAL) InviteForToken(ctx context.Context, token string) (*models.Invite, error) {
	r := dl.Expector.Record(token)
	return r[0].(*models.Invite), mock.SafeError(r[1])
}

func (dl *mockDAL) InvitesForParkedEntityID(ctx context.Context, parkedEntityID string) ([]*models.Invite, error) {
	r := dl.Expector.Record(parkedEntityID)
	if len(r) == 0 {
		return nil, nil
	}
	return r[0].([]*models.Invite), mock.SafeError(r[1])
}

func (dl *mockDAL) DeleteInvite(ctx context.Context, token string) error {
	r := dl.Expector.Record(token)
	return mock.SafeError(r[0])
}

func (dl *mockDAL) InsertEntityToken(ctx context.Context, entityID, token string) error {
	r := dl.Expector.Record(entityID, token)
	return mock.SafeError(r[0])
}

func (dl *mockDAL) TokensForEntity(ctx context.Context, entityID string) ([]string, error) {
	r := dl.Expector.Record(entityID)
	return r[0].([]string), mock.SafeError(r[1])
}

func (dl *mockDAL) UpdateInvite(ctx context.Context, token string, update *models.InviteUpdate) (*models.Invite, error) {
	r := dl.Expector.Record(token, update)
	return r[0].(*models.Invite), mock.SafeError(r[1])
}

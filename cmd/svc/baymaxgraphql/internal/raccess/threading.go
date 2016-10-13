package raccess

import (
	"context"

	"github.com/sprucehealth/backend/svc/threading"
)

func (m *resourceAccessor) CreateSavedMessage(ctx context.Context, orgID string, req *threading.CreateSavedMessageRequest) (*threading.CreateSavedMessageResponse, error) {
	if err := m.canAccessResource(ctx, orgID, m.orgsForOrganization); err != nil {
		return nil, err
	}
	return m.threading.CreateSavedMessage(ctx, req)
}

func (m *resourceAccessor) DeleteSavedMessage(ctx context.Context, req *threading.DeleteSavedMessageRequest) (*threading.DeleteSavedMessageResponse, error) {
	return m.threading.DeleteSavedMessage(ctx, req)
}

func (m *resourceAccessor) SavedMessages(ctx context.Context, req *threading.SavedMessagesRequest) (*threading.SavedMessagesResponse, error) {
	return m.threading.SavedMessages(ctx, req)
}

func (m *resourceAccessor) UpdateSavedMessage(ctx context.Context, req *threading.UpdateSavedMessageRequest) (*threading.UpdateSavedMessageResponse, error) {
	return m.threading.UpdateSavedMessage(ctx, req)
}

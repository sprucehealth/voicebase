package mock

import (
	"context"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/threading"
)

func (m *ResourceAccessor) CreateSavedMessage(ctx context.Context, orgID string, req *threading.CreateSavedMessageRequest) (*threading.CreateSavedMessageResponse, error) {
	rets := m.Record(orgID, req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CreateSavedMessageResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) DeleteSavedMessage(ctx context.Context, req *threading.DeleteSavedMessageRequest) (*threading.DeleteSavedMessageResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.DeleteSavedMessageResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) SavedMessages(ctx context.Context, req *threading.SavedMessagesRequest) (*threading.SavedMessagesResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.SavedMessagesResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) UpdateSavedMessage(ctx context.Context, req *threading.UpdateSavedMessageRequest) (*threading.UpdateSavedMessageResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.UpdateSavedMessageResponse), mock.SafeError(rets[1])
}

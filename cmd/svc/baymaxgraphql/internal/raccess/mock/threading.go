package mock

import (
	"context"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/threading"
)

func (m *ResourceAccessor) Tags(ctx context.Context, req *threading.TagsRequest) (*threading.TagsResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*threading.TagsResponse), mock.SafeError(rets[1])
}

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

func (m *ResourceAccessor) CreateScheduledMessage(ctx context.Context, req *threading.CreateScheduledMessageRequest) (*threading.CreateScheduledMessageResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CreateScheduledMessageResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) DeleteScheduledMessage(ctx context.Context, req *threading.DeleteScheduledMessageRequest) (*threading.DeleteScheduledMessageResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.DeleteScheduledMessageResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) ScheduledMessages(ctx context.Context, req *threading.ScheduledMessagesRequest) (*threading.ScheduledMessagesResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.ScheduledMessagesResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CloneAttachments(ctx context.Context, req *threading.CloneAttachmentsRequest) (*threading.CloneAttachmentsResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CloneAttachmentsResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) BatchJobs(ctx context.Context, req *threading.BatchJobsRequest) (*threading.BatchJobsResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.BatchJobsResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) BatchPostMessages(ctx context.Context, req *threading.BatchPostMessagesRequest) (*threading.BatchPostMessagesResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.BatchPostMessagesResponse), mock.SafeError(rets[1])
}

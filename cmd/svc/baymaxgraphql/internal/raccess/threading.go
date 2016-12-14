package raccess

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/threading"
)

// Saved Messages
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

// Scheduled Messages
func (m *resourceAccessor) CreateScheduledMessage(ctx context.Context, req *threading.CreateScheduledMessageRequest) (*threading.CreateScheduledMessageResponse, error) {
	if !m.isAccountType(ctx, auth.AccountType_PROVIDER) {
		return nil, errors.ErrNotAuthorized(ctx, req.ThreadID)
	}
	if err := m.CanPostMessage(ctx, req.ThreadID); err != nil {
		return nil, errors.Trace(err)
	}
	return m.threading.CreateScheduledMessage(ctx, req)
}

func (m *resourceAccessor) DeleteScheduledMessage(ctx context.Context, req *threading.DeleteScheduledMessageRequest) (*threading.DeleteScheduledMessageResponse, error) {
	if !m.isAccountType(ctx, auth.AccountType_PROVIDER) {
		return nil, errors.ErrNotAuthorized(ctx, req.ScheduledMessageID)
	}
	res, err := m.threading.ScheduledMessages(ctx, &threading.ScheduledMessagesRequest{
		LookupKey: &threading.ScheduledMessagesRequest_ScheduledMessageID{
			ScheduledMessageID: req.ScheduledMessageID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(res.ScheduledMessages) == 0 {
		return &threading.DeleteScheduledMessageResponse{}, nil
	}
	if err := m.CanPostMessage(ctx, res.ScheduledMessages[0].ThreadID); err != nil {
		return nil, errors.Trace(err)
	}
	return m.threading.DeleteScheduledMessage(ctx, req)
}

func (m *resourceAccessor) ScheduledMessages(ctx context.Context, req *threading.ScheduledMessagesRequest) (*threading.ScheduledMessagesResponse, error) {
	if !m.isAccountType(ctx, auth.AccountType_PROVIDER) {
		return nil, errors.ErrNotAuthorized(ctx, req.GetThreadID()+req.GetScheduledMessageID())
	}
	res, err := m.threading.ScheduledMessages(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	switch req.LookupKey.(type) {
	case *threading.ScheduledMessagesRequest_ThreadID:
		if err := m.CanPostMessage(ctx, req.GetThreadID()); err != nil {
			return nil, errors.Trace(err)
		}
	case *threading.ScheduledMessagesRequest_ScheduledMessageID:
		if len(res.ScheduledMessages) == 0 {
			return &threading.ScheduledMessagesResponse{}, nil
		}
		if err := m.CanPostMessage(ctx, res.ScheduledMessages[0].ThreadID); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return res, nil
}

func (m *resourceAccessor) CloneAttachments(ctx context.Context, req *threading.CloneAttachmentsRequest) (*threading.CloneAttachmentsResponse, error) {
	resp, err := m.threading.CloneAttachments(ctx, req)
	return resp, errors.Trace(err)
}

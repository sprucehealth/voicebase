package raccess

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/threading"
)

func (m *resourceAccessor) Tags(ctx context.Context, req *threading.TagsRequest) (*threading.TagsResponse, error) {
	if err := m.canAccessResource(ctx, req.OrganizationID, m.orgsForOrganization); err != nil {
		return nil, err
	}
	return m.threading.Tags(ctx, req)
}

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

func (m *resourceAccessor) BatchJobs(ctx context.Context, req *threading.BatchJobsRequest) (*threading.BatchJobsResponse, error) {
	resp, err := m.threading.BatchJobs(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var entityToAssert string
	switch key := req.LookupKey.(type) {
	case *threading.BatchJobsRequest_ID:
		if len(resp.BatchJobs) != 0 {
			entityToAssert = resp.BatchJobs[0].RequestingEntity
		}
	case *threading.BatchJobsRequest_RequestingEntity:
		entityToAssert = key.RequestingEntity
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown lookup key type")
	}
	// Only the person who requested it should be able to see the status of a batch job for now
	if _, err := m.AssertIsEntity(ctx, entityToAssert); err != nil {
		return nil, errors.Trace(err)
	}
	return resp, nil
}

func (m *resourceAccessor) BatchPostMessages(ctx context.Context, req *threading.BatchPostMessagesRequest) (*threading.BatchPostMessagesResponse, error) {
	// TODO: Figure out how to maybe do this a bit more efficiently. Lave for now since ACL service incoming
	parallel := conc.NewParallel()
	for _, pmr := range req.PostMessagesRequests {
		parallel.Go(func() error {
			return m.CanPostMessage(ctx, pmr.ThreadID)
		})
	}
	if err := parallel.Wait(); err != nil {
		return nil, errors.Trace(err)
	}
	resp, err := m.threading.BatchPostMessages(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return resp, nil
}

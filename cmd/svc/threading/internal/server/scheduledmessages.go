package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
)

func (s *threadsServer) CreateScheduledMessage(ctx context.Context, in *threading.CreateScheduledMessageRequest) (*threading.CreateScheduledMessageResponse, error) {
	if in.ThreadID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID %s", in.ThreadID)
	}
	if in.ActorEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ActorEntityID is required")
	}
	if in.ScheduledFor == 0 {
		return nil, grpcErrorf(codes.InvalidArgument, "ScheduledFor is required")
	}
	scheduledFor := time.Unix(int64(in.ScheduledFor), 0)
	if scheduledFor.Before(s.clk.Now()) {
		return nil, grpcErrorf(codes.InvalidArgument, "ScheduledFor cannot be in the past")
	}

	// Make sure the thread exists
	if threads, err := s.dal.Threads(ctx, []models.ThreadID{threadID}); err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpcErrorf(codes.NotFound, "Thread %s not found", threadID)
	}

	req, err := createPostMessageRequest(ctx, threadID, in.ActorEntityID, in.GetMessage())
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err := claimAttachments(ctx, s.mediaClient, s.paymentsClient, threadID, req.Attachments); err != nil {
		return nil, errors.Trace(err)
	}
	threadItem, err := dal.ThreadItemFromPostMessageRequest(ctx, req, s.clk)
	if err != nil {
		return nil, errors.Trace(err)
	}
	threadItem.ID = models.EmptyThreadItemID()
	data := threadItem.Data.(*models.Message)

	var rScheduledMessage *threading.ScheduledMessage
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		// Transactionally schedule the message, read it back, and return it to the user
		scheduledMessageID, err := dl.CreateScheduledMessage(ctx, &models.ScheduledMessage{
			Type:          models.ItemTypeMessage,
			ScheduledFor:  scheduledFor,
			ActorEntityID: in.ActorEntityID,
			ThreadID:      threadID,
			Internal:      threadItem.Internal,
			Data:          data,
			Status:        models.ScheduledMessageStatusPending, // All scheduled messages start pending
		})
		if err != nil {
			return errors.Trace(err)
		}

		scheduledMessage, err := dl.ScheduledMessage(ctx, scheduledMessageID)
		if err != nil {
			return errors.Trace(err)
		}

		rScheduledMessage, err = transformScheduledMessageToResponse(scheduledMessage)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.CreateScheduledMessageResponse{
		ScheduledMessage: rScheduledMessage,
	}, nil
}

func (s *threadsServer) DeleteScheduledMessage(ctx context.Context, in *threading.DeleteScheduledMessageRequest) (*threading.DeleteScheduledMessageResponse, error) {
	if in.ScheduledMessageID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ScheduledMessageID is required")
	}
	scheduledMessageID, err := models.ParseScheduledMessageID(in.ScheduledMessageID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ScheduledMessageID %s", in.ScheduledMessageID)
	}
	deletedStatus := models.ScheduledMessageStatusDeleted
	if _, err := s.dal.UpdateScheduledMessage(ctx, scheduledMessageID, &models.ScheduledMessageUpdate{
		Status: &deletedStatus,
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.DeleteScheduledMessageResponse{}, nil
}

func (s *threadsServer) ScheduledMessages(ctx context.Context, in *threading.ScheduledMessagesRequest) (*threading.ScheduledMessagesResponse, error) {
	scheduledMessageStatus := make([]models.ScheduledMessageStatus, len(in.Status))
	for i, s := range in.Status {
		status, err := models.ParseScheduledMessageStatus(s.String())
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Unknown ScheduledMessage Status %s", s)
		}
		scheduledMessageStatus[i] = status
	}
	var scheduledMessages []*models.ScheduledMessage
	switch in.LookupKey.(type) {
	case *threading.ScheduledMessagesRequest_ScheduledMessageID:
		scheduledMessageID, err := models.ParseScheduledMessageID(in.GetScheduledMessageID())
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Invalid ScheduledMessageID %s", in.GetScheduledMessageID())
		}
		scheduledMessage, err := s.dal.ScheduledMessage(ctx, scheduledMessageID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found %s", in.GetScheduledMessageID())
		} else if err != nil {
			return nil, errors.Trace(err)
		}
		scheduledMessages = []*models.ScheduledMessage{scheduledMessage}
	case *threading.ScheduledMessagesRequest_ThreadID:
		if in.GetThreadID() == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "ThreadID is required")
		}
		threadID, err := models.ParseThreadID(in.GetThreadID())
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID %s", in.GetThreadID())
		}
		if _, err := s.dal.Threads(ctx, []models.ThreadID{threadID}); errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found %s", threadID)
		} else if err != nil {
			return nil, errors.Trace(err)
		}
		scheduledMessages, err = s.dal.ScheduledMessagesForThread(ctx, threadID, scheduledMessageStatus)
		if err != nil {
			return nil, errors.Trace(err)
		}
	default:
		return nil, grpcErrorf(codes.InvalidArgument, "Unknown Lookup Key Type %s", in.LookupKey)
	}
	rScheduledMessages, err := transformScheduledMessagesToResponse(scheduledMessages)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.ScheduledMessagesResponse{
		ScheduledMessages: rScheduledMessages,
	}, nil
}

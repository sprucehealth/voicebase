package server

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *threadsServer) CreateTriggeredMessage(ctx context.Context, in *threading.CreateTriggeredMessageRequest) (*threading.CreateTriggeredMessageResponse, error) {
	if in.Key == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Key is required")
	}
	if in.Key.Key == threading.TRIGGERED_MESSAGE_KEY_INVALID {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid triggered message key %s", in.Key.Key)
	}
	if in.ActorEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ActorEntityID is required")
	}
	if in.OrganizationEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationEntityID is required")
	}
	if len(in.Messages) == 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "At least 1 Message is required")
	}
	var rtm *threading.TriggeredMessage
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		key, err := transformTriggeredMessageKeyToModel(in.Key.Key)
		if err != nil {
			return errors.Trace(err)
		}

		// Delete existing triggered messages for this set
		tm, err := dl.TriggeredMessageForKeys(ctx, key, in.Key.Subkey)
		if err != nil && errors.Cause(err) != dal.ErrNotFound {
			return errors.Trace(err)
		}

		if tm != nil {
			if err := deleteTriggeredMessage(ctx, tm.ID, dl); err != nil {
				return errors.Trace(err)
			}
		}

		k, err := transformTriggeredMessageKeyToModel(in.Key.Key)
		if err != nil {
			return errors.Trace(err)
		}

		// Insert the new triggered message record and then it's associated message items
		tmID, err := dl.CreateTriggeredMessage(ctx, &models.TriggeredMessage{
			ActorEntityID:        in.ActorEntityID,
			OrganizationEntityID: in.OrganizationEntityID,
			TriggerKey:           k,
			TriggerSubkey:        in.Key.Subkey,
			Enabled:              in.Enabled,
		})
		if err != nil {
			return errors.Trace(err)
		}
		// Insert the associated triggered message item records preserving order
		for i, m := range in.Messages {
			req, err := createPostMessageRequest(ctx, models.EmptyThreadID(), in.ActorEntityID, m)
			if err != nil {
				return errors.Trace(err)
			}
			threadItem, err := dal.ThreadItemFromPostMessageRequest(ctx, req, s.clk)
			if err != nil {
				return errors.Trace(err)
			}
			threadItem.ID = models.EmptyThreadItemID()
			data := threadItem.Data.(*models.Message)

			// Get the attachments out of the message
			attachments, err := transformAttachmentsFromRequest(m.Attachments)
			if err != nil {
				return errors.Trace(err)
			}
			mediaIDs := mediaIDsFromAttachments(attachments)

			// Claim any media attachments for the message
			if len(mediaIDs) > 0 {
				_, err = s.mediaClient.ClaimMedia(ctx, &media.ClaimMediaRequest{
					MediaIDs:  mediaIDs,
					OwnerType: media.MediaOwnerType_TRIGGERED_MESSAGE,
					OwnerID:   tmID.String(),
				})
				if err != nil {
					return errors.Trace(err)
				}
			}

			// Insert the triggered message item
			if _, err := dl.CreateTriggeredMessageItem(ctx, &models.TriggeredMessageItem{
				TriggeredMessageID: tmID,
				Ordinal:            int64(i),
				Internal:           m.Internal,
				Data:               data,
			}); err != nil {
				return errors.Trace(err)
			}
		}

		// Read the information back for response
		rtm, err = triggeredMessage(ctx, tmID, dl)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.CreateTriggeredMessageResponse{
		TriggeredMessage: rtm,
	}, nil
}

func triggeredMessage(ctx context.Context, id models.TriggeredMessageID, dl dal.DAL) (*threading.TriggeredMessage, error) {
	tm, err := dl.TriggeredMessage(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	tmis, err := dl.TriggeredMessageItemsForTriggeredMessage(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	rtm, err := transformTriggeredMessageToResponse(tm, tmis)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return rtm, nil
}

func (s *threadsServer) TriggeredMessages(ctx context.Context, in *threading.TriggeredMessagesRequest) (*threading.TriggeredMessagesResponse, error) {
	var rtms []*threading.TriggeredMessage
	switch lk := in.LookupKey.(type) {
	case (*threading.TriggeredMessagesRequest_Key):
		key, err := transformTriggeredMessageKeyToModel(lk.Key.Key)
		if err != nil {
			return nil, errors.Trace(err)
		}
		rtm, err := triggeredMessageForKeys(ctx, key, lk.Key.Subkey, s.dal)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpc.Errorf(codes.NotFound, "TriggeredMessage not found for key(s) %s %s", key, lk.Key.Subkey)
		} else if err != nil {
			return nil, errors.Trace(err)
		}
		rtms = []*threading.TriggeredMessage{rtm}
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown LookupKey %s", in.LookupKey)
	}
	return &threading.TriggeredMessagesResponse{
		TriggeredMessages: rtms,
	}, nil
}

func triggeredMessageForKeys(ctx context.Context, key, subkey string, dl dal.DAL) (*threading.TriggeredMessage, error) {
	tm, err := dl.TriggeredMessageForKeys(ctx, key, subkey)
	if err != nil {
		return nil, errors.Trace(err)
	}
	tmis, err := dl.TriggeredMessageItemsForTriggeredMessage(ctx, tm.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	rtm, err := transformTriggeredMessageToResponse(tm, tmis)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return rtm, nil
}

func (s *threadsServer) DeleteTriggeredMessage(ctx context.Context, in *threading.DeleteTriggeredMessageRequest) (*threading.DeleteTriggeredMessageResponse, error) {
	if in.TriggeredMessageID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "TriggeredMessageID is required")
	}
	tmID, err := models.ParseTriggeredMessageID(in.TriggeredMessageID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid TriggeredMessageID %s", in.TriggeredMessageID)
	}
	if err := deleteTriggeredMessage(ctx, tmID, s.dal); err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.DeleteTriggeredMessageResponse{}, nil
}

func deleteTriggeredMessage(ctx context.Context, id models.TriggeredMessageID, dl dal.DAL) error {
	return errors.Trace(dl.Transact(ctx, func(ctx context.Context, tdl dal.DAL) error {
		// Clean up associated message items
		if _, err := dl.DeleteTriggeredMessageItemsForTriggeredMessage(ctx, id); err != nil {
			return errors.Trace(err)
		}
		// Clean up the old message
		if _, err := dl.DeleteTriggeredMessage(ctx, id); err != nil {
			return errors.Trace(err)
		}
		return nil
	}))
}

func (s *threadsServer) UpdateTriggeredMessage(ctx context.Context, in *threading.UpdateTriggeredMessageRequest) (*threading.UpdateTriggeredMessageResponse, error) {
	if in.TriggeredMessageID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "TriggeredMessageID is required")
	}
	tmID, err := models.ParseTriggeredMessageID(in.TriggeredMessageID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid TriggeredMessageID %s", in.TriggeredMessageID)
	}
	var rtm *threading.TriggeredMessage
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		// Make sure it exists and lock the row we're updating
		if _, err := dl.TriggeredMessage(ctx, tmID, dal.ForUpdate); errors.Cause(err) == dal.ErrNotFound {
			return grpc.Errorf(codes.NotFound, "TriggeredMessage not found %s", tmID)
		} else if err != nil {
			return errors.Trace(err)
		}

		tmUpdate := &models.TriggeredMessageUpdate{}
		if in.UpdateEnabled {
			tmUpdate.Enabled = ptr.Bool(in.Enabled)
		}
		if _, err := dl.UpdateTriggeredMessage(ctx, tmID, tmUpdate); err != nil {
			return errors.Trace(err)
		}

		// Read the information back for response
		rtm, err = triggeredMessage(ctx, tmID, dl)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.UpdateTriggeredMessageResponse{
		TriggeredMessage: rtm,
	}, nil
}

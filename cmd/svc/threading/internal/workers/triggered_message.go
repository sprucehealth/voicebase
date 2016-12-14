package workers

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/server"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

type triggeredMessageThreadClient interface {
	cloneAttachmentsThreadClient
	postMessagesThreadClient
}

type cloneAttachmentsThreadClient interface {
	CloneAttachments(ctx context.Context, req *threading.CloneAttachmentsRequest, opts ...grpc.CallOption) (*threading.CloneAttachmentsResponse, error)
}

type postMessagesThreadClient interface {
	PostMessages(ctx context.Context, req *threading.PostMessagesRequest, opts ...grpc.CallOption) (*threading.PostMessagesResponse, error)
}

// triggeredMessageItemsWithClonedAttachments fetches the triggered message and prepares the message by cloning all the attachments
func triggeredMessageItemsWithClonedAttachments(
	ctx context.Context,
	dl dal.DAL,
	threadClient cloneAttachmentsThreadClient,
	tmID models.TriggeredMessageID,
	ownerType threading.CloneAttachmentsRequest_OwnerType,
	ownerID string) ([]*models.TriggeredMessageItem, error) {
	triggeredMessageItems, err := dl.TriggeredMessageItemsForTriggeredMessage(ctx, tmID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	parallel := conc.NewParallel()
	for _, item := range triggeredMessageItems {
		switch data := item.Data.(type) {
		case *models.Message:
			// capture
			cData := data
			parallel.Go(func() error {
				msg, err := server.TransformMessageToResponse(cData, false)
				if err != nil {
					return errors.Trace(err)
				}
				if len(msg.Attachments) != 0 {
					resp, err := threadClient.CloneAttachments(ctx, &threading.CloneAttachmentsRequest{
						Attachments: msg.Attachments,
						OwnerType:   ownerType,
						OwnerID:     ownerID,
					})
					if err != nil {
						return errors.Trace(err)
					}
					attModels, err := server.TransformAttachmentsFromRequest(resp.Attachments)
					if err != nil {
						return errors.Trace(err)
					}
					cData.Attachments = attModels
				}
				return nil
			})
		}
	}
	if err := parallel.Wait(); err != nil {
		return nil, errors.Trace(err)
	}
	return triggeredMessageItems, nil
}

func postMessagesForTriggeredMessage(
	ctx context.Context,
	dl dal.DAL,
	threadClient triggeredMessageThreadClient,
	threadID models.ThreadID,
	orgID, key, subkey string) error {
	triggeredMessage, err := dl.TriggeredMessageForKeys(ctx, orgID, key, subkey)
	if errors.Cause(err) == dal.ErrNotFound {
		golog.Debugf("No Triggered Message found for OrgID: %s Key: %s Subkey: %s", orgID, key, subkey)
		return nil
	} else if err != nil {
		return errors.Trace(err)
	}
	if triggeredMessage.Enabled {
		triggeredMessageItems, err := triggeredMessageItemsWithClonedAttachments(
			ctx,
			dl,
			threadClient,
			triggeredMessage.ID,
			threading.CLONED_ATTACHMENT_OWNER_THREAD,
			threadID.String())
		if err != nil {
			return errors.Trace(err)
		}
		messages := make([]*threading.MessagePost, len(triggeredMessageItems))
		// TODO: Make it so this could be from multiple entities - This involves modifying PostMessages to take a message map
		var fromEntityID string
		for i, tmi := range triggeredMessageItems {
			// Decode the message to build the request we'll use to post the mesage
			message, err := server.TransformMessageToResponse(tmi.Data.(*models.Message), false)
			if err != nil {
				return errors.Trace(err)
			}
			messages[i] = &threading.MessagePost{
				Internal:     tmi.Internal,
				Source:       message.Source,
				Destinations: message.Destinations,
				Text:         message.Text,
				Attachments:  message.Attachments,
				Title:        message.Title,
				Summary:      message.Summary,
			}
			fromEntityID = tmi.ActorEntityID
		}
		if len(messages) != 0 {
			if _, err = threadClient.PostMessages(ctx, &threading.PostMessagesRequest{
				UUID:         triggeredMessage.ID.String() + ":" + threadID.String(),
				ThreadID:     threadID.String(),
				FromEntityID: fromEntityID,
				Messages:     messages,
			}); err != nil {
				return errors.Trace(err)
			}
		} else {
			golog.Errorf("Ended up with 0 messages to send after transformation for OrgID: %s Key: %s Subkey: %s", orgID, key, subkey)
			return nil
		}
	} else {
		golog.Debugf("No Welcome Message sent for OrgID: %s Key: %s Subkey: %s - it is currently DISABLED", orgID, key, subkey)
		return nil
	}
	return nil
}

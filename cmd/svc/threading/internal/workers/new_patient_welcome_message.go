package workers

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/server"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

type newPatientWelcomeMessageThreadClient interface {
	PostMessages(ctx context.Context, req *threading.PostMessagesRequest, opts ...grpc.CallOption) (*threading.PostMessagesResponse, error)
}

func (s *Subscriber) newPatientWelcomeMessage(u events.Unmarshaler) error {
	npev, ok := u.(*threading.NewThreadEvent)
	if !ok {
		return errors.Errorf("Expected event of type *threading.NewThreadEvent but got %+v", u)
	}
	return processNewPatientWelcomeMessage(context.Background(), s.dal, s.directoryClient, s.threadClient, npev)
}

func processNewPatientWelcomeMessage(ctx context.Context, dl dal.DAL, directoryClient directory.DirectoryClient, threadClient newPatientWelcomeMessageThreadClient, ntev *threading.NewThreadEvent) error {
	threadID, err := models.ParseThreadID(ntev.ThreadID)
	if err != nil {
		return errors.Wrapf(err, "Invalid Thread ID in New Thread Event %s", ntev.ThreadID)
	}
	threads, err := dl.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return errors.Wrapf(err, "Error while looking up thread from New Thread Event for %s", ntev.ThreadID)
	}
	if len(threads) != 1 {
		return errors.Errorf("Expected to find a single thread for New Thread Event %s bit got %d", ntev.ThreadID, len(threads))
	}
	thread := threads[0]
	if thread.Deleted {
		golog.Warningf("Ignoring Welcome Message for New Thread Event on DELETED thread %v", thread)
		return nil
	}
	if thread.PrimaryEntityID == "" {
		golog.Debugf("No primary entity assigned to thread %s. Ignoring Welcome Message", thread.ID)
		return nil
	}
	entResp, err := directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: thread.PrimaryEntityID,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "Error while looking up primary entity %s for thread %s", thread.PrimaryEntityID, thread.ID)
	} else if len(entResp.Entities) != 1 {
		return errors.Errorf("Error while looking up primary entity %s for thread %s - expected 1 entity but got %d", thread.PrimaryEntityID, thread.ID, len(entResp.Entities))
	}
	ent := entResp.Entities[0]
	if ent.Source == nil {
		golog.Debugf("No source for entity %v - ignoring welcome message", ent.ID)
	}
	subkey := directory.FlattenEntitySource(ent.Source)
	triggeredMessage, err := dl.TriggeredMessageForKeys(ctx, models.TriggeredMessageKeyNewPatient, subkey)
	if errors.Cause(err) == dal.ErrNotFound {
		golog.Debugf("No Welcome Message found for Key: %s Subkey: %s", models.TriggeredMessageKeyNewPatient, subkey)
		return nil
	} else if err != nil {
		return errors.Trace(err)
	}
	if triggeredMessage.Enabled {
		triggeredMessageItems, err := dl.TriggeredMessageItemsForTriggeredMessage(ctx, triggeredMessage.ID)
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
		if _, err = threadClient.PostMessages(ctx, &threading.PostMessagesRequest{
			UUID:         triggeredMessage.ID.String() + ":" + ntev.ThreadID,
			ThreadID:     ntev.ThreadID,
			FromEntityID: fromEntityID,
			Messages:     messages,
		}); err != nil {
			return errors.Trace(err)
		}
	} else {
		golog.Debugf("No Welcome Message SENT for Key: %s Subkey: %s sing it is currently DISABLED", models.TriggeredMessageKeyNewPatient, subkey)
	}
	return nil
}

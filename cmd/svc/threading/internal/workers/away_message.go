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
)

func (s *Subscriber) awayMessage(u events.Unmarshaler) error {
	ptiev, ok := u.(*threading.PublishedThreadItem)
	if !ok {
		return errors.Errorf("Expected event of type *threading.PublishedThreadItem but got %+v", u)
	}
	return processAwayMessage(context.Background(), s.dal, s.directoryClient, s.threadClient, ptiev)
}

func processAwayMessage(ctx context.Context, dl dal.DAL, directoryClient directory.DirectoryClient, threadClient triggeredMessageThreadClient, ptiev *threading.PublishedThreadItem) error {
	if ptiev.Item.GetMessage() == nil {
		golog.Debugf("Ignoring away message for non Message event %+v")
		return nil
	}
	threadID, err := models.ParseThreadID(ptiev.ThreadID)
	if err != nil {
		return errors.Wrapf(err, "Invalid Thread ID in Published Thread Item Event %s", ptiev.ThreadID)
	}
	threads, err := dl.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return errors.Wrapf(err, "Error while looking up thread from Published Thead Item Event for %s", ptiev.ThreadID)
	}
	if len(threads) != 1 {
		return errors.Errorf("Expected to find a single thread for Published Thead Item Event %s bit got %d", ptiev.ThreadID, len(threads))
	}
	thread := threads[0]
	if thread.Deleted {
		golog.Warningf("Ignoring Published Thead Item Event for Away Message on DELETED thread %v", thread)
		return nil
	}
	entResp, err := directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: ptiev.Item.ActorEntityID,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "Error while looking up actor entity %s for thread %s", ptiev.Item.ActorEntityID, thread.ID)
	} else if len(entResp.Entities) != 1 {
		return errors.Errorf("Error while looking up actor entity %s for thread %s - expected 1 entity but got %d", ptiev.Item.ActorEntityID, thread.ID, len(entResp.Entities))
	}
	ent := entResp.Entities[0]

	thType, err := server.TransformThreadTypeToResponse(thread.Type)
	if err != nil {
		return errors.Errorf("Unable to transform model thread type %s to response", thread.Type)
	}
	var destinations []*threading.Endpoint
	var channel *threading.Endpoint_Channel
	if ptiev.Item.GetMessage().Source != nil {
		destinations = []*threading.Endpoint{ptiev.Item.GetMessage().Source}
		channel = &ptiev.Item.GetMessage().Source.Channel
	}
	return errors.Trace(postMessagesForTriggeredMessage(
		ctx,
		dl,
		threadClient,
		thread.ID,
		destinations,
		thread.OrganizationID,
		models.TriggeredMessageKeyAwayMessage,
		threading.AwayMessageSubkey(ent.Type, thType, channel),
	))
}

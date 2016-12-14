package workers

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/threading"
)

func (s *Subscriber) newPatientWelcomeMessage(u events.Unmarshaler) error {
	npev, ok := u.(*threading.NewThreadEvent)
	if !ok {
		return errors.Errorf("Expected event of type *threading.NewThreadEvent but got %+v", u)
	}
	return processNewPatientWelcomeMessage(context.Background(), s.dal, s.directoryClient, s.threadClient, npev)
}

func processNewPatientWelcomeMessage(ctx context.Context, dl dal.DAL, directoryClient directory.DirectoryClient, threadClient triggeredMessageThreadClient, ntev *threading.NewThreadEvent) error {
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
		return nil
	}
	return errors.Trace(postMessagesForTriggeredMessage(
		ctx,
		dl,
		threadClient,
		thread.ID,
		thread.OrganizationID,
		models.TriggeredMessageKeyNewPatient,
		threading.WelcomeMessageSubkey(ent.Source),
	))
}

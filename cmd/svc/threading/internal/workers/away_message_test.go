package workers

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestAwayMessage(t *testing.T) {
	tID, err := models.NewThreadID()
	test.OK(t, err)
	t.Run("Ignore-NonMessage", func(t *testing.T) {
		st := newSubscriptionsTest(t)
		defer st.Finish()
		testAwayMessage(t, st, &threading.PublishedThreadItem{Item: &threading.ThreadItem{}}, nil)
	})
	t.Run("Ignore-DeletedThread", func(t *testing.T) {
		st := newSubscriptionsTest(t)
		defer st.Finish()
		st.dal.Expect(mock.NewExpectation(st.dal.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
			{Deleted: true},
		}, nil))
		testAwayMessage(t, st, &threading.PublishedThreadItem{
			ThreadID: tID.String(),
			Item: &threading.ThreadItem{
				Item: &threading.ThreadItem_Message{
					Message: &threading.Message{},
				},
			},
		}, nil)
	})
	t.Run("Ignore-NoAwayMessage", func(t *testing.T) {
		st := newSubscriptionsTest(t)
		defer st.Finish()
		st.dal.Expect(mock.NewExpectation(st.dal.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
			{
				ID:             tID,
				Type:           models.ThreadTypeSecureExternal,
				OrganizationID: "OrganizationID",
			},
		}, nil))
		st.directoryClient.EXPECT().LookupEntities(st.ctx, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_EntityID{
				EntityID: "ActorEntityID",
			},
		}).Return(&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_PATIENT,
				},
			},
		}, nil)
		st.dal.Expect(mock.NewExpectation(
			st.dal.TriggeredMessageForKeys, "OrganizationID", models.TriggeredMessageKeyNewPatient, "PATIENT:THREAD_TYPE_SECURE_EXTERNAL", []interface{}{}).WithReturns(
			(*models.TriggeredMessage)(nil), dal.ErrNotFound))
		testAwayMessage(t, st, &threading.PublishedThreadItem{
			ThreadID: tID.String(),
			Item: &threading.ThreadItem{
				ActorEntityID: "ActorEntityID",
				Item: &threading.ThreadItem_Message{
					Message: &threading.Message{},
				},
			},
		}, nil)
	})
}

func testAwayMessage(
	t *testing.T,
	st *subscriptionsTest,
	in *threading.PublishedThreadItem,
	expErr error) {
	test.Equals(t, expErr, processAwayMessage(st.ctx, st.dal, st.directoryClient, st.threadsClient, in))
}

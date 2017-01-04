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

func TestNewPatientWelcomeMessage(t *testing.T) {
	tID, err := models.NewThreadID()
	test.OK(t, err)
	t.Run("Ignore-NoThread", func(t *testing.T) {
		st := newSubscriptionsTest(t)
		defer st.Finish()
		st.dal.Expect(mock.NewExpectation(st.dal.Threads, []models.ThreadID{tID}).WithReturns(([]*models.Thread)(nil), nil))
		testNewPatientWelcomeMessage(t, st, &threading.NewThreadEvent{
			ThreadID: tID.String(),
		}, nil)
	})
	t.Run("Ignore-NoPrimaryEntity", func(t *testing.T) {
		st := newSubscriptionsTest(t)
		defer st.Finish()
		st.dal.Expect(mock.NewExpectation(st.dal.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
			{PrimaryEntityID: ""},
		}, nil))
		testNewPatientWelcomeMessage(t, st, &threading.NewThreadEvent{
			ThreadID: tID.String(),
		}, nil)
	})
	t.Run("Ignore-NoEntitySource", func(t *testing.T) {
		st := newSubscriptionsTest(t)
		defer st.Finish()
		st.dal.Expect(mock.NewExpectation(st.dal.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
			{PrimaryEntityID: "primaryEntityID"},
		}, nil))
		st.directoryClient.EXPECT().LookupEntities(st.ctx, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_EntityID{
				EntityID: "primaryEntityID",
			},
		}).Return(&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{},
			},
		}, nil)
		testNewPatientWelcomeMessage(t, st, &threading.NewThreadEvent{
			ThreadID: tID.String(),
		}, nil)
	})
	t.Run("Ignore-NoWelcomeMessage", func(t *testing.T) {
		st := newSubscriptionsTest(t)
		defer st.Finish()
		st.dal.Expect(mock.NewExpectation(st.dal.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
			{
				PrimaryEntityID: "primaryEntityID",
				OrganizationID:  "OrganizationID",
			},
		}, nil))
		st.directoryClient.EXPECT().LookupEntities(st.ctx, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_EntityID{
				EntityID: "primaryEntityID",
			},
		}).Return(&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Source: &directory.EntitySource{
						Type: directory.EntitySource_PRACTICE_CODE,
						Data: "123456",
					},
				},
			},
		}, nil)
		st.dal.Expect(mock.NewExpectation(
			st.dal.TriggeredMessageForKeys, "OrganizationID", models.TriggeredMessageKeyNewPatient, "PRACTICE_CODE:123456", []interface{}{}).WithReturns(
			(*models.TriggeredMessage)(nil), dal.ErrNotFound))
		testNewPatientWelcomeMessage(t, st, &threading.NewThreadEvent{
			ThreadID: tID.String(),
		}, nil)
	})
}

func testNewPatientWelcomeMessage(
	t *testing.T,
	st *subscriptionsTest,
	in *threading.NewThreadEvent,
	expErr error) {
	test.Equals(t, expErr, processNewPatientWelcomeMessage(st.ctx, st.dal, st.directoryClient, st.threadsClient, in))
}

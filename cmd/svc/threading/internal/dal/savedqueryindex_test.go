package dal

import (
	"context"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

func TestSavedQueryIndex(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	sq := &models.SavedQuery{
		Ordinal:  1,
		EntityID: "ent",
		Query: &models.Query{
			Expressions: []*models.Expr{
				{Value: &models.Expr_Token{Token: "summary"}},
			},
		},
		Title: "sq1",
	}
	_, err := dal.CreateSavedQuery(ctx, sq)
	test.OK(t, err)

	sqr, err := dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, sq, sqr)

	// Create some threads
	t1 := &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeExternal,
		LastMessageSummary:         "thread1 summary",
		LastMessageTimestamp:       time.Unix(10e8, 0),
		LastExternalMessageSummary: "extsummary",
	}
	tid1, err := dal.CreateThread(ctx, t1)
	test.OK(t, err)
	test.OK(t, dal.AddThreadMembers(ctx, tid1, []string{"org"}))
	t2 := &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "thread2 summary",
		LastMessageTimestamp:       time.Unix(11e8, 0),
		LastExternalMessageSummary: "extsummary",
	}
	tid2, err := dal.CreateThread(ctx, t2)
	test.OK(t, err)

	// Add a unread thread

	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq.ID, ThreadID: tid1, Unread: true, Timestamp: t1.LastMessageTimestamp},
	}))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 1, sqr.Total)

	// Add a read thread

	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq.ID, ThreadID: tid2, Unread: false, Timestamp: t2.LastMessageTimestamp},
	}))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 2, sqr.Total)

	// Update unread thread to read

	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq.ID, ThreadID: tid1, Unread: false, Timestamp: t1.LastMessageTimestamp},
	}))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 0, sqr.Unread)
	test.Equals(t, 2, sqr.Total)

	// Update read thread to unread

	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq.ID, ThreadID: tid2, Unread: true, Timestamp: t2.LastMessageTimestamp.Add(time.Second)},
	}))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 2, sqr.Total)

	// Delete read thread

	test.OK(t, dal.RemoveItemsFromSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq.ID, ThreadID: tid1},
	}))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 1, sqr.Total)

	// Delete unread thread

	test.OK(t, dal.RemoveItemsFromSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq.ID, ThreadID: tid2},
	}))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 0, sqr.Unread)
	test.Equals(t, 0, sqr.Total)

	// One on read thread and one unread thread

	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq.ID, ThreadID: tid1, Unread: false, Timestamp: t1.LastMessageTimestamp},
		{SavedQueryID: sq.ID, ThreadID: tid2, Unread: true, Timestamp: t2.LastMessageTimestamp},
	}))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 2, sqr.Total)

	// Iterate threads in saved query

	tc, err := dal.IterateThreadsInSavedQuery(ctx, sq.ID, "ent", &Iterator{Direction: FromStart})
	test.OK(t, err)
	test.Equals(t, 2, len(tc.Edges))
	test.Equals(t, false, tc.HasMore)
	test.Equals(t, "thread2 summary", tc.Edges[0].Thread.LastMessageSummary)
	test.Equals(t, "thread1 summary", tc.Edges[1].Thread.LastMessageSummary)

	// Delete all threads in saved query

	test.OK(t, dal.RemoveAllItemsFromSavedQueryIndex(ctx, sq.ID))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 0, sqr.Unread)
	test.Equals(t, 0, sqr.Total)

	// Add an unread thread

	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq.ID, ThreadID: tid2, Unread: true, Timestamp: t2.LastMessageTimestamp},
	}))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 1, sqr.Total)

	// Delete all thread from all saved queries

	test.OK(t, dal.RemoveThreadFromAllSavedQueryIndexes(ctx, tid2))
	sqr, err = dal.SavedQuery(ctx, sq.ID)
	test.OK(t, err)
	test.Equals(t, 0, sqr.Unread)
	test.Equals(t, 0, sqr.Total)
}

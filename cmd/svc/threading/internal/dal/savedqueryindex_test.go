package dal

import (
	"context"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

func TestSavedQueryIndex(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB, clock.New())
	ctx := context.Background()

	sq := &models.SavedQuery{
		Ordinal:  1,
		EntityID: "ent",
		Query: &models.Query{
			Expressions: []*models.Expr{
				{Value: &models.Expr_Token{Token: "summary"}},
			},
		},
		ShortTitle:           "sq1",
		NotificationsEnabled: true,
		Type:                 models.SavedQueryTypeNormal,
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

	// Add an unread thread

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

func TestLargeBatchSavedQueryItems(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB, clock.New())
	ctx := context.Background()

	sq := &models.SavedQuery{
		Ordinal:  1,
		EntityID: "ent",
		Query: &models.Query{
			Expressions: []*models.Expr{
				{Value: &models.Expr_Token{Token: "summary"}},
			},
		},
		ShortTitle:           "sq1",
		NotificationsEnabled: true,
		Type:                 models.SavedQueryTypeNormal,
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

	items := make([]*SavedQueryThread, 5000)
	for i := range items {
		items[i] = &SavedQueryThread{ThreadID: tid1, SavedQueryID: sqr.ID, Timestamp: time.Now(), Unread: false}
	}
	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, items))
	test.OK(t, dal.RemoveItemsFromSavedQueryIndex(ctx, items))
}

func TestNotificationsSavedQuery(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB, clock.New())
	ctx := context.Background()

	sq1 := &models.SavedQuery{
		Ordinal:  1,
		EntityID: "ent",
		Query: &models.Query{
			Expressions: []*models.Expr{
				{Value: &models.Expr_Token{Token: "summary"}},
			},
		},
		ShortTitle:           "sq1",
		NotificationsEnabled: true,
		Type:                 models.SavedQueryTypeNormal,
	}
	_, err := dal.CreateSavedQuery(ctx, sq1)
	test.OK(t, err)

	sq2 := &models.SavedQuery{
		Ordinal:  2,
		EntityID: "ent",
		Query: &models.Query{
			Expressions: []*models.Expr{
				{Value: &models.Expr_Token{Token: "summary"}},
			},
		},
		ShortTitle:           "sq1",
		NotificationsEnabled: true,
		Type:                 models.SavedQueryTypeNormal,
	}
	_, err = dal.CreateSavedQuery(ctx, sq2)
	test.OK(t, err)

	nsq := &models.SavedQuery{
		Ordinal:              3,
		EntityID:             "ent",
		Query:                &models.Query{},
		ShortTitle:           "nsq",
		NotificationsEnabled: false,
		Type:                 models.SavedQueryTypeNotifications,
	}
	_, err = dal.CreateSavedQuery(ctx, nsq)
	test.OK(t, err)

	sqr, err := dal.SavedQuery(ctx, sq1.ID)
	test.OK(t, err)
	test.Equals(t, sq1, sqr)

	sqr, err = dal.SavedQuery(ctx, sq2.ID)
	test.OK(t, err)
	test.Equals(t, sq2, sqr)

	sqr, err = dal.SavedQuery(ctx, nsq.ID)
	test.OK(t, err)
	test.Equals(t, nsq, sqr)

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

	counts, err := dal.UnreadNotificationsCounts(ctx, []string{"ent"})
	test.OK(t, err)
	test.Equals(t, map[string]int{"ent": 0}, counts)

	// Add an unread thread

	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq1.ID, ThreadID: tid1, Unread: true, Timestamp: t1.LastMessageTimestamp},
	}))
	sqr, err = dal.SavedQuery(ctx, sq1.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 1, sqr.Total)

	test.OK(t, dal.RebuildNotificationsSavedQuery(ctx, "ent"))
	sqr, err = dal.SavedQuery(ctx, nsq.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 1, sqr.Total)

	counts, err = dal.UnreadNotificationsCounts(ctx, []string{"ent"})
	test.OK(t, err)
	test.Equals(t, map[string]int{"ent": 1}, counts)

	// Add same thread to second saved query

	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq2.ID, ThreadID: tid1, Unread: true, Timestamp: t1.LastMessageTimestamp},
	}))
	sqr, err = dal.SavedQuery(ctx, sq2.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 1, sqr.Total)

	test.OK(t, dal.RebuildNotificationsSavedQuery(ctx, "ent"))
	sqr, err = dal.SavedQuery(ctx, nsq.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 1, sqr.Total)

	counts, err = dal.UnreadNotificationsCounts(ctx, []string{"ent"})
	test.OK(t, err)
	test.Equals(t, map[string]int{"ent": 1}, counts)

	// Add read thread

	test.OK(t, dal.AddItemsToSavedQueryIndex(ctx, []*SavedQueryThread{
		{SavedQueryID: sq2.ID, ThreadID: tid2, Unread: false, Timestamp: t2.LastMessageTimestamp},
	}))
	sqr, err = dal.SavedQuery(ctx, sq2.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 2, sqr.Total)

	test.OK(t, dal.RebuildNotificationsSavedQuery(ctx, "ent"))
	sqr, err = dal.SavedQuery(ctx, nsq.ID)
	test.OK(t, err)
	test.Equals(t, 1, sqr.Unread)
	test.Equals(t, 2, sqr.Total)

	counts, err = dal.UnreadNotificationsCounts(ctx, []string{"ent"})
	test.OK(t, err)
	test.Equals(t, map[string]int{"ent": 1}, counts)
}

func TestNotificationCountPrimaryEntity(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB, clock.New())
	ctx := context.Background()

	// Create a thread
	thread := &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeExternal,
		LastMessageSummary:         "thread1 summary",
		LastMessageTimestamp:       time.Unix(10e8, 0),
		LastExternalMessageSummary: "extsummary",
		PrimaryEntityID:            "ext1",
	}
	tid, err := dal.CreateThread(ctx, thread)
	test.OK(t, err)

	counts, err := dal.UnreadNotificationsCounts(ctx, []string{"nonexistant"})
	test.OK(t, err)
	test.Equals(t, map[string]int{}, counts)

	_, err = dal.PostMessage(ctx, &PostMessageRequest{
		ThreadID:     tid,
		FromEntityID: "actor",
		Title:        "external",
		Text:         "text",
		Summary:      "summary",
	})
	test.OK(t, err)

	counts, err = dal.UnreadNotificationsCounts(ctx, []string{"ext1"})
	test.OK(t, err)
	test.Equals(t, map[string]int{"ext1": 1}, counts)

	_, err = dal.PostMessage(ctx, &PostMessageRequest{
		ThreadID:     tid,
		FromEntityID: "actor",
		Title:        "internal",
		Text:         "text",
		Summary:      "summary",
		Internal:     true,
	})
	test.OK(t, err)

	counts, err = dal.UnreadNotificationsCounts(ctx, []string{"ext1"})
	test.OK(t, err)
	test.Equals(t, map[string]int{"ext1": 1}, counts)
}

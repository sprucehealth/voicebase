package dal

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

const schemaGlob = "../../schema/*.sql"

func TestTimeCursor(t *testing.T) {
	t.Parallel()
	tm := time.Unix(1, 234567890)

	// sanity check mainly for documentation purposes
	test.Equals(t, int64(1234567890), tm.UnixNano())

	// should return the time in microsecond
	ms := formatTimeCursor(tm)
	test.Equals(t, "1234567", ms)

	tm2, err := parseTimeCursor(ms)
	test.OK(t, err)
	test.Equals(t, int64(1234567000), tm2.UnixNano())
}

func TestDedupeStrings(t *testing.T) {
	test.Equals(t, []string(nil), dedupeStrings(nil))
	test.Equals(t, []string{"a"}, dedupeStrings([]string{"a"}))
	test.Equals(t, []string{"a"}, dedupeStrings([]string{"a", "a"}))
	test.Equals(t, []string{"a", "b"}, dedupeStrings([]string{"a", "b"}))
	test.Equals(t, []string{"a", "c", "b"}, dedupeStrings([]string{"a", "a", "b", "c"}))
	test.Equals(t, []string{"a", "b", "c"}, dedupeStrings([]string{"a", "b", "b", "c"}))
	test.Equals(t, []string{"a", "b", "c"}, dedupeStrings([]string{"a", "b", "c", "c"}))
	test.Equals(t, []string{"a", "b", "c"}, dedupeStrings([]string{"a", "b", "c", "a"}))
}

func TestTransact(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		var tid models.ThreadID
		var terr error
		err := dal.Transact(ctx, func(ctx context.Context, dl DAL) error {
			tid, terr = dl.CreateThread(ctx, &models.Thread{
				OrganizationID:             "org",
				Type:                       models.ThreadTypeExternal,
				LastMessageSummary:         "summary",
				LastMessageTimestamp:       time.Unix(10e8, 0),
				LastExternalMessageSummary: "extsummary",
			})
			return terr
		})
		test.OK(t, terr)
		test.OK(t, err)

		ts, err := dal.Threads(ctx, []models.ThreadID{tid})
		test.OK(t, err)
		test.Equals(t, 1, len(ts))
	})

	t.Run("fail", func(t *testing.T) {
		var tid models.ThreadID
		var terr error
		err := dal.Transact(ctx, func(ctx context.Context, dl DAL) error {
			tid, terr = dl.CreateThread(ctx, &models.Thread{
				OrganizationID:             "org",
				Type:                       models.ThreadTypeExternal,
				LastMessageSummary:         "summary",
				LastMessageTimestamp:       time.Unix(10e8, 0),
				LastExternalMessageSummary: "extsummary",
			})
			return errors.New("FAIL")
		})
		test.OK(t, terr)
		test.Assert(t, err != nil, "Err should not be nil on transaction error")
		test.Equals(t, "FAIL", errors.Cause(err).Error())

		ts, err := dal.Threads(ctx, []models.ThreadID{tid})
		test.OK(t, err)
		test.Equals(t, 0, len(ts))
	})

	t.Run("panic", func(t *testing.T) {
		var tid models.ThreadID
		var terr error
		err := dal.Transact(ctx, func(ctx context.Context, dl DAL) error {
			tid, terr = dl.CreateThread(ctx, &models.Thread{
				OrganizationID:             "org",
				Type:                       models.ThreadTypeExternal,
				LastMessageSummary:         "summary",
				LastMessageTimestamp:       time.Unix(10e8, 0),
				LastExternalMessageSummary: "extsummary",
			})
			panic("BOOM")
		})
		test.OK(t, terr)
		test.Assert(t, err != nil, "Err should not be nil on panic")
		test.Equals(t, "Encountered panic during transaction execution: BOOM", errors.Cause(err).Error())

		ts, err := dal.Threads(ctx, []models.ThreadID{tid})
		test.OK(t, err)
		test.Equals(t, 0, len(ts))
	})
}

func TestIterateThreads(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	// Create external thread
	tid1, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeExternal,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(10e8, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)
	test.OK(t, dal.AddThreadMembers(ctx, tid1, []string{"org"}))
	// Create team thread
	tid2, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(11e8, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	// Viewer without membership in team thread should only see external thread
	tc, err := dal.IterateThreads(ctx, nil, []string{"org", "viewer"}, "viewer", false, &Iterator{
		Direction: FromStart,
		Count:     10,
	})
	test.OK(t, err)
	t.Logf("%+v", tc)
	test.Equals(t, 1, len(tc.Edges))
	test.Equals(t, tid1, tc.Edges[0].Thread.ID)
	test.Equals(t, (*models.ThreadEntity)(nil), tc.Edges[0].ThreadEntity)

	// Still not a member but now has a thread entity row
	test.OK(t, dal.UpdateThreadEntity(ctx, tid2, "viewer", nil))
	tc, err = dal.IterateThreads(ctx, nil, []string{"org", "viewer"}, "viewer", false, &Iterator{
		Direction: FromStart,
		Count:     10,
	})
	test.OK(t, err)
	t.Logf("%+v", tc)
	test.Equals(t, 1, len(tc.Edges))
	test.Equals(t, tid1, tc.Edges[0].Thread.ID)
	test.Equals(t, (*models.ThreadEntity)(nil), tc.Edges[0].ThreadEntity)

	// Now they're a member and should get both threads
	test.OK(t, dal.AddThreadMembers(ctx, tid2, []string{"viewer"}))
	tc, err = dal.IterateThreads(ctx, nil, []string{"org", "viewer"}, "viewer", false, &Iterator{
		Direction: FromStart,
		Count:     10,
	})
	test.OK(t, err)
	t.Logf("%+v", tc)
	test.Equals(t, 2, len(tc.Edges))
	test.Equals(t, tid2, tc.Edges[0].Thread.ID)
	test.Equals(t, tid1, tc.Edges[1].Thread.ID)
	test.Equals(t, &models.ThreadEntity{
		ThreadID: tid2,
		EntityID: "viewer",
		Member:   true,
		Joined:   tc.Edges[0].ThreadEntity.Joined,
	}, tc.Edges[0].ThreadEntity)
	test.Equals(t, (*models.ThreadEntity)(nil), tc.Edges[1].ThreadEntity)

	// Make sure we don't get duplicates if both org and viewer are members
	test.OK(t, dal.AddThreadMembers(ctx, tid2, []string{"org"}))
	tc, err = dal.IterateThreads(ctx, nil, []string{"org", "viewer"}, "viewer", false, &Iterator{
		Direction: FromStart,
		Count:     10,
	})
	test.OK(t, err)
	t.Logf("%+v", tc)
	test.Equals(t, 2, len(tc.Edges))
	test.Equals(t, tid2, tc.Edges[0].Thread.ID)
	test.Equals(t, tid1, tc.Edges[1].Thread.ID)
	test.Equals(t, &models.ThreadEntity{
		ThreadID: tid2,
		EntityID: "viewer",
		Member:   true,
		Joined:   tc.Edges[0].ThreadEntity.Joined,
	}, tc.Edges[0].ThreadEntity)
	test.Equals(t, (*models.ThreadEntity)(nil), tc.Edges[1].ThreadEntity)
}

func TestIterateThreadsQuery(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	// Create external thread
	tid1, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeExternal,
		SystemTitle:                "Zoe Smith",
		LastMessageSummary:         "Some message or other with the patient",
		LastMessageTimestamp:       time.Unix(10e8, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)
	test.OK(t, dal.AddThreadMembers(ctx, tid1, []string{"org"}))
	// Create team thread
	tid2, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		UserTitle:                  "User set title",
		LastMessageSummary:         "Blah blah foo other bar",
		LastMessageTimestamp:       time.Unix(11e8, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)
	test.OK(t, dal.AddThreadMembers(ctx, tid2, []string{"viewer"}))
	// Create team thread 2
	tid3, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		UserTitle:                  "User",
		LastMessageSummary:         "Summary",
		LastMessageTimestamp:       time.Unix(12e8, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)
	test.OK(t, dal.AddThreadMembers(ctx, tid3, []string{"viewer"}))

	// Thread 1 has been read
	test.OK(t, dal.UpdateThreadEntity(ctx, tid1, "viewer", &ThreadEntityUpdate{
		LastViewed: ptr.Time(time.Unix(10e8, 0)),
	}))
	// Thread 2 has a reference, has not been read (no thread entity record)
	test.OK(t, dal.UpdateThreadEntity(ctx, tid2, "viewer", &ThreadEntityUpdate{
		LastReferenced: ptr.Time(time.Unix(10e8, 0)),
	}))
	// Thread 3 has a reference, has not been read, but the reference is already read
	test.OK(t, dal.UpdateThreadEntity(ctx, tid3, "viewer", &ThreadEntityUpdate{
		LastViewed:     ptr.Time(time.Unix(11e8, 0)),
		LastReferenced: ptr.Time(time.Unix(10e8, 0)),
	}))

	cases := map[string]*struct {
		query *models.Query
		ids   []models.ThreadID
	}{
		"non-existant-token": {
			query: &models.Query{
				Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "Nonexistant"}}},
			},
			ids: []models.ThreadID{},
		},
		"token-single-match": {
			query: &models.Query{
				Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "zoe"}}},
			},
			ids: []models.ThreadID{tid1},
		},
		"token-multiple-match": {
			query: &models.Query{
				Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "OTHER"}}},
			},
			ids: []models.ThreadID{tid2, tid1},
		},
		"type-patient": {
			query: &models.Query{
				Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_PATIENT}}},
			},
			ids: []models.ThreadID{tid1},
		},
		"type-team": {
			query: &models.Query{
				Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_TEAM}}},
			},
			ids: []models.ThreadID{tid3, tid2},
		},
		"flag-unread": {
			query: &models.Query{
				Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}},
			},
			ids: []models.ThreadID{tid3, tid2},
		},
		"flag-referenced": {
			query: &models.Query{
				Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD_REFERENCE}}},
			},
			ids: []models.ThreadID{tid2},
		},
	}

	for tcName, tc := range cases {
		t.Run(tcName, func(t *testing.T) {
			con, err := dal.IterateThreads(ctx, tc.query, []string{"org", "viewer"}, "viewer", false, &Iterator{
				Direction: FromStart,
				Count:     10,
			})
			test.OK(t, err)
			test.Equals(t, len(tc.ids), len(con.Edges))
			for i, id := range tc.ids {
				test.Equals(t, id, con.Edges[i].Thread.ID)
			}
		})
	}
}

func TestThread(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
		SystemTitle:                "systemTitle",
		UserTitle:                  "userTitle",
	})
	test.OK(t, err)

	ths, err := dal.Threads(ctx, []models.ThreadID{tid})
	test.OK(t, err)
	test.Equals(t, 1, len(ths))
	th := ths[0]
	test.Equals(t, "systemTitle", th.SystemTitle)
	test.Equals(t, "userTitle", th.UserTitle)

	// for update
	ths, err = dal.Threads(ctx, []models.ThreadID{tid}, ForUpdate)
	test.OK(t, err)
	test.Equals(t, 1, len(ths))
	th = ths[0]
	test.Equals(t, "systemTitle", th.SystemTitle)
	test.Equals(t, "userTitle", th.UserTitle)
}

func TestThreadsWithEntity(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
		SystemTitle:                "systemTitle",
		UserTitle:                  "userTitle",
	})
	test.OK(t, err)

	ths, tes, err := dal.ThreadsWithEntity(ctx, "ent", []models.ThreadID{tid})
	test.OK(t, err)
	test.Equals(t, 1, len(ths))
	test.Equals(t, 1, len(tes))
	test.Equals(t, (*models.ThreadEntity)(nil), tes[0])

	test.OK(t, dal.UpdateThreadEntity(ctx, tid, "ent", &ThreadEntityUpdate{}))

	ths, tes, err = dal.ThreadsWithEntity(ctx, "ent", []models.ThreadID{tid})
	test.OK(t, err)
	test.Equals(t, 1, len(ths))
	test.Equals(t, 1, len(tes))
	test.Equals(t, &models.ThreadEntity{
		ThreadID: tid,
		EntityID: "ent",
		Joined:   tes[0].Joined,
	}, tes[0])
}

func TestUpdateThread(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
		SystemTitle:                "systemTitle",
		UserTitle:                  "userTitle",
	})
	test.OK(t, err)

	ths, err := dal.Threads(ctx, []models.ThreadID{tid})
	test.OK(t, err)
	test.Equals(t, 1, len(ths))
	th := ths[0]
	test.Equals(t, "systemTitle", th.SystemTitle)
	test.Equals(t, "userTitle", th.UserTitle)

	test.OK(t, dal.UpdateThread(ctx, tid, &ThreadUpdate{SystemTitle: ptr.String("foo"), UserTitle: ptr.String("bar")}))

	ths, err = dal.Threads(ctx, []models.ThreadID{tid})
	test.OK(t, err)
	test.Equals(t, 1, len(ths))
	th = ths[0]
	test.OK(t, err)
	test.Equals(t, "foo", th.SystemTitle)
	test.Equals(t, "bar", th.UserTitle)
}

func TestThreadEntities(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	ents, err := dal.ThreadEntities(ctx, []models.ThreadID{tid}, "e1")
	test.OK(t, err)
	test.Equals(t, 0, len(ents))

	test.OK(t, dal.UpdateThreadEntity(ctx, tid, "e1", &ThreadEntityUpdate{LastViewed: ptr.Time(time.Unix(1e6, 0))}))

	ents, err = dal.ThreadEntities(ctx, []models.ThreadID{tid}, "e1", ForUpdate)
	test.OK(t, err)
	test.Equals(t, 1, len(ents))
	test.Equals(t, time.Unix(1e6, 0), *ents[tid.String()].LastViewed)
	test.Equals(t, (*time.Time)(nil), ents[tid.String()].LastReferenced)

	test.OK(t, dal.UpdateThreadEntity(ctx, tid, "e1", &ThreadEntityUpdate{LastReferenced: ptr.Time(time.Unix(1e6, 0))}))

	ents, err = dal.ThreadEntities(ctx, []models.ThreadID{tid}, "e1", ForUpdate)
	test.OK(t, err)
	test.Equals(t, 1, len(ents))
	test.Equals(t, time.Unix(1e6, 0), *ents[tid.String()].LastReferenced)
}

func TestThreadsForOrg(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	_, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org1",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	tid2, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org1",
		Type:                       models.ThreadTypeSupport,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	_, err = dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org2",
		Type:                       models.ThreadTypeSupport,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	threads, err := dal.ThreadsForOrg(ctx, "org1", "", 10)
	test.OK(t, err)
	test.Equals(t, 2, len(threads))
	for _, th := range threads {
		test.Equals(t, th.OrganizationID, "org1")
	}

	threads, err = dal.ThreadsForOrg(ctx, "org1", models.ThreadTypeSupport, 10)
	test.OK(t, err)
	test.Equals(t, 1, len(threads))
	test.Equals(t, threads[0].ID, tid2)
	test.Equals(t, threads[0].OrganizationID, "org1")
	test.Equals(t, threads[0].Type, models.ThreadTypeSupport)
}

func TestAddRemoveThreadMembers(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	test.OK(t, dal.AddThreadMembers(ctx, tid, []string{"e1"}))
	tes, err := dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	test.Equals(t, 1, len(tes))
	test.Equals(t, tes[0].EntityID, "e1")
	test.Equals(t, tes[0].Member, true)

	test.OK(t, dal.AddThreadMembers(ctx, tid, []string{"e2"}))
	tes, err = dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	sort.Sort(teByID(tes))
	test.Equals(t, 2, len(tes))
	test.Equals(t, tes[0].EntityID, "e1")
	test.Equals(t, tes[0].Member, true)
	test.Equals(t, tes[1].EntityID, "e2")
	test.Equals(t, tes[1].Member, true)

	test.OK(t, dal.AddThreadMembers(ctx, tid, []string{"e1", "e2"}))
	tes, err = dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	sort.Sort(teByID(tes))
	test.Equals(t, 2, len(tes))
	test.Equals(t, tes[0].EntityID, "e1")
	test.Equals(t, tes[0].Member, true)
	test.Equals(t, tes[1].EntityID, "e2")
	test.Equals(t, tes[1].Member, true)

	test.OK(t, dal.RemoveThreadMembers(ctx, tid, []string{"e2"}))
	test.OK(t, dal.AddThreadMembers(ctx, tid, []string{"e3"}))
	tes, err = dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	sort.Sort(teByID(tes))
	test.Equals(t, 3, len(tes))
	test.Equals(t, tes[0].EntityID, "e1")
	test.Equals(t, tes[0].Member, true)
	test.Equals(t, tes[1].EntityID, "e2")
	test.Equals(t, tes[1].Member, false)
	test.Equals(t, tes[2].EntityID, "e3")
	test.Equals(t, tes[2].Member, true)
}

func TestSetupThreadState(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeSetup,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)
	test.OK(t, dal.CreateSetupThreadState(ctx, tid, "ent"))

	state, err := dal.SetupThreadState(ctx, tid)
	test.OK(t, err)
	test.Equals(t, tid, state.ThreadID)
	test.Equals(t, 0, state.Step)

	state, err = dal.SetupThreadStateForEntity(ctx, "ent")
	test.OK(t, err)
	test.Equals(t, tid, state.ThreadID)
	test.Equals(t, 0, state.Step)

	test.OK(t, dal.UpdateSetupThreadState(ctx, tid, &SetupThreadStateUpdate{Step: ptr.Int(1)}))

	state, err = dal.SetupThreadState(ctx, tid, ForUpdate)
	test.OK(t, err)
	test.Equals(t, tid, state.ThreadID)
	test.Equals(t, 1, state.Step)
}

func TestDeleteThread(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
		SystemTitle:                "systemTitle",
		UserTitle:                  "userTitle",
	})
	test.OK(t, err)

	ts, err := dal.Threads(ctx, []models.ThreadID{tid})
	test.OK(t, err)
	test.Equals(t, 1, len(ts))

	test.OK(t, dal.DeleteThread(ctx, tid))

	ts, err = dal.Threads(ctx, []models.ThreadID{tid})
	test.OK(t, err)
	test.Equals(t, 0, len(ts))
}

func TestFollowers(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
		SystemTitle:                "systemTitle",
		UserTitle:                  "userTitle",
	})
	test.OK(t, err)

	tes, err := dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	test.Equals(t, 0, len(tes))

	test.OK(t, dal.AddThreadFollowers(ctx, tid, []string{"ent"}))

	tes, err = dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	test.Equals(t, 1, len(tes))
	test.Equals(t, true, tes[0].Following)

	test.OK(t, dal.RemoveThreadFollowers(ctx, tid, []string{"ent"}))

	tes, err = dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	test.Equals(t, 1, len(tes))
	test.Equals(t, false, tes[0].Following)
}

func TestCreateThreadItemViewDetails(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(1e9, 0),
		LastExternalMessageSummary: "extsummary",
		SystemTitle:                "systemTitle",
		UserTitle:                  "userTitle",
	})
	test.OK(t, err)

	ti, err := dal.PostMessage(ctx, &PostMessageRequest{
		ThreadID:     tid,
		FromEntityID: "actor",
		Title:        "title",
		Text:         "text",
		Summary:      "summary",
	})
	test.OK(t, err)

	details := []*models.ThreadItemViewDetails{
		{
			ThreadItemID:  ti.ID,
			ActorEntityID: "actor",
			ViewTime:      ptr.Time(time.Now()),
		},
	}
	test.OK(t, dal.CreateThreadItemViewDetails(ctx, details))
	// Duplicate details should be ignore
	test.OK(t, dal.CreateThreadItemViewDetails(ctx, details))
}

func TestSavedQueries(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	sq1 := &models.SavedQuery{
		Ordinal:              2,
		EntityID:             "ent",
		Query:                &models.Query{},
		Title:                "sq1",
		NotificationsEnabled: true,
	}
	_, err := dal.CreateSavedQuery(ctx, sq1)
	test.OK(t, err)

	sq2 := &models.SavedQuery{
		Ordinal:  1,
		EntityID: "ent",
		Query: &models.Query{
			Expressions: []*models.Expr{
				{Value: &models.Expr_Token{Token: "foo"}},
			},
		},
		Title:                "sq2",
		NotificationsEnabled: true,
	}
	_, err = dal.CreateSavedQuery(ctx, sq2)
	test.OK(t, err)

	sq, err := dal.SavedQuery(ctx, sq1.ID)
	test.OK(t, err)
	test.Equals(t, sq1, sq)
	sq, err = dal.SavedQuery(ctx, sq2.ID)
	test.OK(t, err)
	test.Equals(t, sq2, sq)

	sqs, err := dal.SavedQueries(ctx, "ent")
	test.OK(t, err)
	test.Equals(t, 2, len(sqs))
	test.Equals(t, sq1.ID, sqs[1].ID)
	test.Equals(t, sq2.ID, sqs[0].ID)

	newQuery := &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD_REFERENCE}}}}
	test.OK(t, dal.UpdateSavedQuery(ctx, sq1.ID, &SavedQueryUpdate{
		Title:   ptr.String("new title"),
		Ordinal: ptr.Int(19),
		Query:   newQuery,
	}))

	sq, err = dal.SavedQuery(ctx, sq1.ID)
	test.OK(t, err)
	test.Equals(t, "new title", sq.Title)
	test.Equals(t, 19, sq.Ordinal)
	test.Equals(t, newQuery, sq.Query)
}

type teByID []*models.ThreadEntity

func (tes teByID) Len() int           { return len(tes) }
func (tes teByID) Swap(a, b int)      { tes[a], tes[b] = tes[b], tes[a] }
func (tes teByID) Less(a, b int) bool { return tes[a].EntityID < tes[b].EntityID }

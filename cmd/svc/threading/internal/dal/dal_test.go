package dal

import (
	"sort"
	"testing"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
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
		LastMessageTimestamp:       time.Unix(1, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)
	test.OK(t, dal.AddThreadMembers(ctx, tid1, []string{"org"}))
	// Create team thread
	tid2, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(2, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	// Viewer without membership in team thread should only see external thread
	tc, err := dal.IterateThreads(ctx, []string{"org", "viewer"}, "viewer", false, &Iterator{
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
	tc, err = dal.IterateThreads(ctx, []string{"org", "viewer"}, "viewer", false, &Iterator{
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
	tc, err = dal.IterateThreads(ctx, []string{"org", "viewer"}, "viewer", false, &Iterator{
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
	tc, err = dal.IterateThreads(ctx, []string{"org", "viewer"}, "viewer", false, &Iterator{
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

func TestThread(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(2, 0),
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

func TestUpdateThread(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(2, 0),
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
		LastMessageTimestamp:       time.Unix(2, 0),
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
		LastMessageTimestamp:       time.Unix(2, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	tid2, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org1",
		Type:                       models.ThreadTypeSupport,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(2, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	_, err = dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org2",
		Type:                       models.ThreadTypeSupport,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(2, 0),
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
		LastMessageTimestamp:       time.Unix(2, 0),
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
		LastMessageTimestamp:       time.Unix(2, 0),
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

func TestCreateThreadItemViewDetails(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(2, 0),
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

type teByID []*models.ThreadEntity

func (tes teByID) Len() int           { return len(tes) }
func (tes teByID) Swap(a, b int)      { tes[a], tes[b] = tes[b], tes[a] }
func (tes teByID) Less(a, b int) bool { return tes[a].EntityID < tes[b].EntityID }

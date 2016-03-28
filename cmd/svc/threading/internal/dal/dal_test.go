package dal

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type dbTest struct {
	db   *sql.DB
	name string
}

func setupTest(t *testing.T) *dbTest {
	user := os.Getenv("TEST_DB_USER")
	if user == "" {
		t.Skip("Missing TEST_DB_USER")
	}

	migrations, err := filepath.Glob("../../schema/*.sql")
	test.OK(t, err)
	sort.Strings(migrations)

	dbName := fmt.Sprintf("test_threading_%d", rand.Int())
	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:     "localhost",
		Name:     "mysql",
		User:     user,
		Password: "",
	})
	test.OK(t, err)
	_, err = db.Exec(`CREATE DATABASE ` + dbName)
	test.OK(t, err)
	_, err = db.Exec(`USE ` + dbName)
	test.OK(t, err)
	for _, m := range migrations {
		b, err := ioutil.ReadFile(m)
		if err != nil {
			db.Exec(`DELETE DATABASE ` + dbName)
			t.Fatal(err)
		}
		s := string(b)
		lines := strings.Split(s, "\n")
		nonEmpty := make([]string, 0, len(lines))
		for _, l := range lines {
			if i := strings.Index(l, "--"); i >= 0 {
				l = l[:i]
			}
			l := strings.TrimSpace(l)
			if l != "" {
				nonEmpty = append(nonEmpty, l)
			}
		}
		stmts := strings.Split(strings.Join(nonEmpty, "\n"), ";")
		for _, st := range stmts {
			st = strings.TrimSpace(st)
			if st != "" {
				if _, err := db.Exec(st); err != nil {
					db.Exec(`DELETE DATABASE ` + dbName)
					t.Fatalf("Failed to apply migration %s: %s\nstatement: %s", m, err, st)
				}
			}
		}
	}
	return &dbTest{
		db:   db,
		name: dbName,
	}
}

func (dt *dbTest) cleanup(t *testing.T) {
	_, err := dt.db.Exec(`DROP DATABASE ` + dt.name)
	if err != nil {
		t.Log(err)
	}
}

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
	dt := setupTest(t)
	defer dt.cleanup(t)

	dal := New(dt.db)
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
	tc, err := dal.IterateThreads(ctx, "org", "viewer", false, &Iterator{
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
	tc, err = dal.IterateThreads(ctx, "org", "viewer", false, &Iterator{
		Direction: FromStart,
		Count:     10,
	})
	test.OK(t, err)
	t.Logf("%+v", tc)
	test.Equals(t, 1, len(tc.Edges))
	test.Equals(t, tid1, tc.Edges[0].Thread.ID)
	test.Equals(t, (*models.ThreadEntity)(nil), tc.Edges[0].ThreadEntity)

	// Now they're a member and should get both threads
	test.OK(t, dal.UpdateThreadEntity(ctx, tid2, "viewer", &ThreadEntityUpdate{Member: ptr.Bool(true)}))
	tc, err = dal.IterateThreads(ctx, "org", "viewer", false, &Iterator{
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
	dt := setupTest(t)
	defer dt.cleanup(t)

	dal := New(dt.db)
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

	th, err := dal.Thread(ctx, tid)
	test.OK(t, err)
	test.Equals(t, "systemTitle", th.SystemTitle)
	test.Equals(t, "userTitle", th.UserTitle)

	// for update
	th, err = dal.Thread(ctx, tid, ForUpdate)
	test.OK(t, err)
	test.Equals(t, "systemTitle", th.SystemTitle)
	test.Equals(t, "userTitle", th.UserTitle)
}

func TestUpdateThread(t *testing.T) {
	dt := setupTest(t)
	defer dt.cleanup(t)

	dal := New(dt.db)
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

	th, err := dal.Thread(ctx, tid)
	test.OK(t, err)
	test.Equals(t, "systemTitle", th.SystemTitle)
	test.Equals(t, "userTitle", th.UserTitle)

	test.OK(t, dal.UpdateThread(ctx, tid, &ThreadUpdate{SystemTitle: ptr.String("foo"), UserTitle: ptr.String("bar")}))

	th, err = dal.Thread(ctx, tid)
	test.OK(t, err)
	test.Equals(t, "foo", th.SystemTitle)
	test.Equals(t, "bar", th.UserTitle)
}

func TestThreadEntities(t *testing.T) {
	dt := setupTest(t)
	defer dt.cleanup(t)

	dal := New(dt.db)
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

func TestUpdateThreadMembers(t *testing.T) {
	dt := setupTest(t)
	defer dt.cleanup(t)

	dal := New(dt.db)
	ctx := context.Background()

	tid, err := dal.CreateThread(ctx, &models.Thread{
		OrganizationID:             "org",
		Type:                       models.ThreadTypeTeam,
		LastMessageSummary:         "summary",
		LastMessageTimestamp:       time.Unix(2, 0),
		LastExternalMessageSummary: "extsummary",
	})
	test.OK(t, err)

	test.OK(t, dal.UpdateThreadMembers(ctx, tid, []string{}))
	tes, err := dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	test.Equals(t, 0, len(tes))

	test.OK(t, dal.UpdateThreadMembers(ctx, tid, []string{"e1"}))
	tes, err = dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	test.Equals(t, 1, len(tes))
	test.Equals(t, tes[0].EntityID, "e1")
	test.Equals(t, tes[0].Member, true)

	test.OK(t, dal.UpdateThreadMembers(ctx, tid, []string{"e2"}))
	tes, err = dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	sort.Sort(teByID(tes))
	test.Equals(t, 2, len(tes))
	test.Equals(t, tes[0].EntityID, "e1")
	test.Equals(t, tes[0].Member, false)
	test.Equals(t, tes[1].EntityID, "e2")
	test.Equals(t, tes[1].Member, true)

	test.OK(t, dal.UpdateThreadMembers(ctx, tid, []string{"e1", "e2"}))
	tes, err = dal.EntitiesForThread(ctx, tid)
	test.OK(t, err)
	sort.Sort(teByID(tes))
	test.Equals(t, 2, len(tes))
	test.Equals(t, tes[0].EntityID, "e1")
	test.Equals(t, tes[0].Member, true)
	test.Equals(t, tes[1].EntityID, "e2")
	test.Equals(t, tes[1].Member, true)

	test.OK(t, dal.UpdateThreadMembers(ctx, tid, []string{"e1", "e1", "e3", "e3"}))
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

type teByID []*models.ThreadEntity

func (tes teByID) Len() int           { return len(tes) }
func (tes teByID) Swap(a, b int)      { tes[a], tes[b] = tes[b], tes[a] }
func (tes teByID) Less(a, b int) bool { return tes[a].EntityID < tes[b].EntityID }

package server

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	mock_directory "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/notification"
	mock_notification "github.com/sprucehealth/backend/svc/notification/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
)

func init() {
	conc.Testing = true
}

func TestCreateSavedQuery(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	eid, err := models.NewSavedQueryID()
	test.OK(t, err)
	esq := &models.SavedQuery{OrganizationID: "o1", EntityID: "e1"}
	dl.Expect(mock.NewExpectation(dl.CreateSavedQuery, esq).WithReturns(eid, nil))
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)
	res, err := srv.CreateSavedQuery(nil, &threading.CreateSavedQueryRequest{
		OrganizationID: "o1",
		EntityID:       "e1",
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateSavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID:             eid.String(),
			OrganizationID: "o1",
		},
	}, res)
}

func TestCreateEmptyThread(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()

	now := time.Unix(1e7, 0)

	thid, err := models.NewThreadID()
	test.OK(t, err)
	th := &models.Thread{
		OrganizationID:     "o1",
		PrimaryEntityID:    "e2",
		LastMessageSummary: "summ",
	}
	dl.Expect(mock.NewExpectation(dl.CreateThread, th).WithReturns(thid, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateMember, thid, "e1", &dal.MemberUpdate{Following: ptr.Bool(true)}).WithReturns(nil))
	th2 := &models.Thread{
		ID:                   thid,
		OrganizationID:       "o1",
		PrimaryEntityID:      "e2",
		LastMessageTimestamp: now,
		LastMessageSummary:   "summ",
		Created:              now,
		MessageCount:         0,
	}
	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(th2, nil))

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)
	res, err := srv.CreateEmptyThread(nil, &threading.CreateEmptyThreadRequest{
		OrganizationID:  "o1",
		FromEntityID:    "e1",
		PrimaryEntityID: "e2",
		Source: &threading.Endpoint{
			ID:      "555-555-5555",
			Channel: threading.Endpoint_SMS,
		},
		Summary: "summ",
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateEmptyThreadResponse{
		Thread: &threading.Thread{
			ID:                         th2.ID.String(),
			OrganizationID:             "o1",
			PrimaryEntityID:            "e2",
			LastMessageTimestamp:       uint64(now.Unix()),
			LastMessageSummary:         "summ",
			LastPrimaryEntityEndpoints: []*threading.Endpoint{},
			CreatedTimestamp:           uint64(now.Unix()),
			MessageCount:               0,
		},
	}, res)
}

func TestCreateThread(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	now := time.Now()

	thid, err := models.NewThreadID()
	test.OK(t, err)
	th := &models.Thread{OrganizationID: "o1", PrimaryEntityID: "e1"}
	dl.Expect(mock.NewExpectation(dl.CreateThread, th).WithReturns(thid, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateMember, thid, "e1", &dal.MemberUpdate{Following: ptr.Bool(true)}).WithReturns(nil))

	mid, err := models.NewThreadItemID()
	test.OK(t, err)
	ps := &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "e1",
		Internal:     true,
		Title:        "foo % woo",
		Text:         "<ref id=\"e2\" type=\"entity\">Foo</ref> bar",
		Attachments:  []*models.Attachment{},
		Source: &models.Endpoint{
			ID:      "555-555-5555",
			Channel: models.Endpoint_SMS,
		},
		TextRefs: []*models.Reference{
			{ID: "e2", Type: models.Reference_ENTITY},
		},
		Summary: "Foo bar",
	}
	ti := &models.ThreadItem{
		ID:            mid,
		ThreadID:      thid,
		Created:       now,
		ActorEntityID: ps.FromEntityID,
		Internal:      ps.Internal,
		Type:          models.ItemTypeMessage,
		Data: &models.Message{
			Title:    ps.Title,
			Text:     ps.Text,
			Status:   models.Message_NORMAL,
			Source:   ps.Source,
			TextRefs: ps.TextRefs,
			Summary:  ps.Summary,
		},
	}
	dl.Expect(mock.NewExpectation(dl.PostMessage, ps).WithReturns(ti, nil))
	th2 := &models.Thread{
		ID:                   thid,
		OrganizationID:       "o1",
		PrimaryEntityID:      "e1",
		LastMessageTimestamp: now,
		LastMessageSummary:   ps.Summary,
		Created:              now,
		MessageCount:         0,
	}
	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(th2, nil))

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)
	res, err := srv.CreateThread(nil, &threading.CreateThreadRequest{
		OrganizationID: "o1",
		FromEntityID:   "e1",
		Title:          "foo % woo",
		Text:           "<ref id=\"e2\" type=\"Entity\">Foo</ref> bar",
		Internal:       true,
		Source: &threading.Endpoint{
			ID:      "555-555-5555",
			Channel: threading.Endpoint_SMS,
		},
		Summary: "Foo bar",
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateThreadResponse{
		ThreadID: thid.String(),
		ThreadItem: &threading.ThreadItem{
			ID:             mid.String(),
			Timestamp:      uint64(now.Unix()),
			Type:           threading.ThreadItem_MESSAGE,
			Internal:       true,
			ActorEntityID:  "e1",
			ThreadID:       th2.ID.String(),
			OrganizationID: "o1",
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:  "foo % woo",
					Text:   "<ref id=\"e2\" type=\"entity\">Foo</ref> bar",
					Status: threading.Message_NORMAL,
					Source: &threading.Endpoint{
						ID:      "555-555-5555",
						Channel: threading.Endpoint_SMS,
					},
					TextRefs: []*threading.Reference{
						{ID: "e2", Type: threading.Reference_ENTITY},
					},
				},
			},
		},
		Thread: &threading.Thread{
			ID:                         th2.ID.String(),
			OrganizationID:             "o1",
			PrimaryEntityID:            "e1",
			LastMessageTimestamp:       uint64(now.Unix()),
			LastMessageSummary:         ps.Summary,
			LastPrimaryEntityEndpoints: []*threading.Endpoint{},
			CreatedTimestamp:           uint64(now.Unix()),
			MessageCount:               0,
		},
	}, res)
}

func TestThreadItem(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)

	eid, err := models.NewThreadItemID()
	test.OK(t, err)
	tid, err := models.NewThreadID()
	test.OK(t, err)
	now := time.Now()
	eti := &models.ThreadItem{
		ID:            eid,
		Type:          models.ItemTypeMessage,
		Created:       now,
		Internal:      true,
		ActorEntityID: "e2",
		ThreadID:      tid,
		Data: &models.Message{
			Title:  "abc",
			Text:   "hello",
			Status: models.Message_NORMAL,
			Source: &models.Endpoint{
				ID:      "555-555-5555",
				Channel: models.Endpoint_VOICE,
			},
			EditedTimestamp: 123,
			EditorEntityID:  "entity:1",
		},
	}
	dl.Expect(mock.NewExpectation(dl.ThreadItem, eid).WithReturns(eti, nil))
	dl.Expect(mock.NewExpectation(dl.Thread, tid).WithReturns(&models.Thread{OrganizationID: "orgID"}, nil))
	res, err := srv.ThreadItem(nil, &threading.ThreadItemRequest{
		ItemID: eid.String(),
	})
	test.OK(t, err)
	test.Equals(t, &threading.ThreadItemResponse{
		Item: &threading.ThreadItem{
			ID:             eid.String(),
			Timestamp:      uint64(now.Unix()),
			Type:           threading.ThreadItem_MESSAGE,
			Internal:       true,
			ActorEntityID:  "e2",
			ThreadID:       tid.String(),
			OrganizationID: "orgID",
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:  "abc",
					Text:   "hello",
					Status: threading.Message_NORMAL,
					Source: &threading.Endpoint{
						ID:      "555-555-5555",
						Channel: threading.Endpoint_VOICE,
					},
					EditedTimestamp: 123,
					EditorEntityID:  "entity:1",
					TextRefs:        []*threading.Reference{},
				},
			},
		},
	}, res)
}

func TestQueryThreads(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)

	orgID := "entity:1"
	peID := "entity:2"
	tID, err := models.NewThreadID()
	test.OK(t, err)
	now := time.Now()
	created := time.Now()

	// Adhoc query

	dl.Expect(mock.NewExpectation(dl.IterateThreads, orgID, false, &dal.Iterator{
		EndCursor: "c1",
		Direction: dal.FromEnd,
		Count:     11,
	}).WithReturns(&dal.ThreadConnection{
		HasMore: true,
		Edges: []dal.ThreadEdge{
			{
				Cursor: "c2",
				Thread: &models.Thread{
					ID:                   tID,
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: now,
					LastPrimaryEntityEndpoints: models.EndpointList{
						Endpoints: []*models.Endpoint{
							{
								Channel: models.Endpoint_SMS,
								ID:      "+1234567890",
							},
						},
					},
					Created:      created,
					MessageCount: 32,
				},
			},
		},
	}, nil))

	res, err := srv.QueryThreads(nil, &threading.QueryThreadsRequest{
		OrganizationID: orgID,
		Iterator: &threading.Iterator{
			EndCursor: "c1",
			Direction: threading.Iterator_FROM_END,
			Count:     11,
		},
		Type: threading.QueryThreadsRequest_ADHOC,
		QueryType: &threading.QueryThreadsRequest_Query{
			Query: &threading.Query{},
		},
	})
	test.OK(t, err)
	test.Equals(t, &threading.QueryThreadsResponse{
		HasMore: true,
		Edges: []*threading.ThreadEdge{
			{
				Thread: &threading.Thread{
					ID:                   tID.String(),
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: uint64(now.Unix()),
					LastPrimaryEntityEndpoints: []*threading.Endpoint{
						{
							Channel: threading.Endpoint_SMS,
							ID:      "+1234567890",
						},
					},
					CreatedTimestamp: uint64(created.Unix()),
					MessageCount:     32,
					Unread:           false,
				},
				Cursor: "c2",
			},
		},
	}, res)
}

func TestQueryThreadsWithViewer(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)

	orgID := "entity:1"
	peID := "entity:2"
	tID, err := models.NewThreadID()
	test.OK(t, err)
	tID2, err := models.NewThreadID()
	test.OK(t, err)
	tID3, err := models.NewThreadID()
	test.OK(t, err)
	now := time.Now()

	// Adhoc query
	dl.Expect(mock.NewExpectation(dl.IterateThreads, orgID, false, &dal.Iterator{
		EndCursor: "c1",
		Direction: dal.FromEnd,
		Count:     11,
	}).WithReturns(&dal.ThreadConnection{
		HasMore: true,
		Edges: []dal.ThreadEdge{
			{
				Cursor: "c2",
				Thread: &models.Thread{
					ID:                   tID,
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: now,
					Created:              time.Unix(now.Unix()-1000, 0),
					MessageCount:         32,
				},
			},
			{
				Cursor: "c3",
				Thread: &models.Thread{
					ID:                   tID2,
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: time.Unix(now.Unix()-1000, 0),
					Created:              time.Unix(now.Unix()-2000, 0),
					MessageCount:         33,
				},
			},
			{
				Cursor: "c4",
				Thread: &models.Thread{
					ID:                   tID3,
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: now,
					Created:              now,
					MessageCount:         0,
				},
			},
		},
	}, nil))

	// Since we have a viewer associated with this query, expect the memberships to be queried to populate read status
	dl.Expect(mock.NewExpectation(dl.ThreadMemberships, []models.ThreadID{tID, tID2, tID3}, peID, false).WithReturns(
		[]*models.ThreadMember{
			{
				ThreadID:   tID,
				EntityID:   peID,
				LastViewed: ptr.Time(time.Unix(1, 1)),
			},
			{
				ThreadID:   tID2,
				EntityID:   peID,
				LastViewed: ptr.Time(now),
			},
			{
				ThreadID:   tID3,
				EntityID:   peID,
				LastViewed: ptr.Time(now),
			},
		}, nil,
	))

	res, err := srv.QueryThreads(nil, &threading.QueryThreadsRequest{
		ViewerEntityID: peID,
		OrganizationID: orgID,
		Iterator: &threading.Iterator{
			EndCursor: "c1",
			Direction: threading.Iterator_FROM_END,
			Count:     11,
		},
		Type: threading.QueryThreadsRequest_ADHOC,
		QueryType: &threading.QueryThreadsRequest_Query{
			Query: &threading.Query{},
		},
	})
	test.OK(t, err)
	test.Equals(t, &threading.QueryThreadsResponse{
		HasMore: true,
		Edges: []*threading.ThreadEdge{
			{
				Thread: &threading.Thread{
					ID:                   tID.String(),
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: uint64(now.Unix()),
					Unread:               true,
					LastPrimaryEntityEndpoints: []*threading.Endpoint{},
					CreatedTimestamp:           uint64(time.Unix(now.Unix()-1000, 0).Unix()),
					MessageCount:               32,
				},
				Cursor: "c2",
			},
			{
				Thread: &threading.Thread{
					ID:                   tID2.String(),
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: uint64(time.Unix(now.Unix()-1000, 0).Unix()),
					Unread:               false,
					LastPrimaryEntityEndpoints: []*threading.Endpoint{},
					CreatedTimestamp:           uint64(time.Unix(now.Unix()-2000, 0).Unix()),
					MessageCount:               33,
				},
				Cursor: "c3",
			},
			{
				Thread: &threading.Thread{
					ID:                   tID3.String(),
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: uint64(now.Unix()),
					Unread:               false,
					LastPrimaryEntityEndpoints: []*threading.Endpoint{},
					CreatedTimestamp:           uint64(now.Unix()),
					MessageCount:               0,
				},
				Cursor: "c4",
			},
		},
	}, res)
}

func TestThread(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)

	thID, err := models.NewThreadID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()
	created := time.Now()

	dl.Expect(mock.NewExpectation(dl.Thread, thID).WithReturns(
		&models.Thread{
			ID:                   thID,
			OrganizationID:       orgID,
			PrimaryEntityID:      entID,
			LastMessageTimestamp: now,
			Created:              created,
			MessageCount:         32,
		}, nil))
	res, err := srv.Thread(nil, &threading.ThreadRequest{
		ThreadID: thID.String(),
	})
	test.OK(t, err)
	test.Equals(t, &threading.ThreadResponse{
		Thread: &threading.Thread{
			ID:                         thID.String(),
			OrganizationID:             orgID,
			PrimaryEntityID:            entID,
			LastMessageTimestamp:       uint64(now.Unix()),
			LastPrimaryEntityEndpoints: []*threading.Endpoint{},
			CreatedTimestamp:           uint64(created.Unix()),
			MessageCount:               32,
			Unread:                     false,
		},
	}, res)
}

func TestThreadWithViewer(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)

	thID, err := models.NewThreadID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()
	created := time.Now()

	dl.Expect(mock.NewExpectation(dl.Thread, thID).WithReturns(
		&models.Thread{
			ID:                   thID,
			OrganizationID:       orgID,
			PrimaryEntityID:      entID,
			LastMessageTimestamp: now,
			Created:              created,
			MessageCount:         32,
		}, nil))
	// Since we have a viewer associated with this query, expect the memberships to be queried to populate read status
	dl.Expect(mock.NewExpectation(dl.ThreadMemberships, []models.ThreadID{thID}, entID, false).WithReturns(
		[]*models.ThreadMember{
			{
				ThreadID:   thID,
				EntityID:   entID,
				LastViewed: ptr.Time(time.Unix(1, 1)),
			},
		}, nil,
	))
	res, err := srv.Thread(nil, &threading.ThreadRequest{
		ThreadID:       thID.String(),
		ViewerEntityID: entID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.ThreadResponse{
		Thread: &threading.Thread{
			ID:                   thID.String(),
			OrganizationID:       orgID,
			PrimaryEntityID:      entID,
			LastMessageTimestamp: uint64(now.Unix()),
			Unread:               true,
			LastPrimaryEntityEndpoints: []*threading.Endpoint{},
			CreatedTimestamp:           uint64(created.Unix()),
			MessageCount:               32,
		},
	}, res)
}

func TestThreadWithViewerNoMembership(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)

	thID, err := models.NewThreadID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()
	created := time.Now()

	dl.Expect(mock.NewExpectation(dl.Thread, thID).WithReturns(
		&models.Thread{
			ID:                   thID,
			OrganizationID:       orgID,
			PrimaryEntityID:      entID,
			LastMessageTimestamp: now,
			Created:              created,
			MessageCount:         32,
		}, nil))
	// Since we have a viewer associated with this query, expect the memberships to be queried and return none, this should be marked as unread
	dl.Expect(mock.NewExpectation(dl.ThreadMemberships, []models.ThreadID{thID}, entID, false))
	res, err := srv.Thread(nil, &threading.ThreadRequest{
		ThreadID:       thID.String(),
		ViewerEntityID: entID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.ThreadResponse{
		Thread: &threading.Thread{
			ID:                   thID.String(),
			OrganizationID:       orgID,
			PrimaryEntityID:      entID,
			LastMessageTimestamp: uint64(now.Unix()),
			Unread:               true,
			LastPrimaryEntityEndpoints: []*threading.Endpoint{},
			CreatedTimestamp:           uint64(created.Unix()),
			MessageCount:               32,
		},
	}, res)
}

func TestThreadWithViewerNoMessages(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)

	thID, err := models.NewThreadID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()
	created := time.Now()

	dl.Expect(mock.NewExpectation(dl.Thread, thID).WithReturns(
		&models.Thread{
			ID:                   thID,
			OrganizationID:       orgID,
			PrimaryEntityID:      entID,
			LastMessageTimestamp: now,
			Created:              created,
			MessageCount:         0,
		}, nil))
	// Since we have a viewer associated with this query, expect the memberships to be queried and return none, this should be marked as unread
	dl.Expect(mock.NewExpectation(dl.ThreadMemberships, []models.ThreadID{thID}, entID, false))
	res, err := srv.Thread(nil, &threading.ThreadRequest{
		ThreadID:       thID.String(),
		ViewerEntityID: entID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.ThreadResponse{
		Thread: &threading.Thread{
			ID:                   thID.String(),
			OrganizationID:       orgID,
			PrimaryEntityID:      entID,
			LastMessageTimestamp: uint64(now.Unix()),
			Unread:               false, // An empty thread should never be unread
			LastPrimaryEntityEndpoints: []*threading.Endpoint{},
			CreatedTimestamp:           uint64(created.Unix()),
			MessageCount:               0,
		},
	}, res)
}

func TestSavedQuery(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil)

	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()

	dl.Expect(mock.NewExpectation(dl.SavedQuery, sqID).WithReturns(
		&models.SavedQuery{
			ID:             sqID,
			OrganizationID: orgID,
			EntityID:       entID,
			Created:        now,
			Modified:       now,
		}, nil))
	res, err := srv.SavedQuery(nil, &threading.SavedQueryRequest{
		SavedQueryID: sqID.String(),
	})
	test.OK(t, err)
	test.Equals(t, &threading.SavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID:             sqID.String(),
			OrganizationID: orgID,
		},
	}, res)
}

func TestMarkThreadAsRead(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID1, err := models.NewThreadItemID()
	test.OK(t, err)
	tiID2, err := models.NewThreadItemID()
	test.OK(t, err)
	eID := "entity:1"
	lView := ptr.Time(time.Unix(time.Now().Unix()-1000, 0))
	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil)

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadMemberships, []models.ThreadID{tID}, eID, true).WithReturns(
		[]*models.ThreadMember{
			{
				ThreadID:   tID,
				EntityID:   eID,
				LastViewed: lView,
			},
		}, nil,
	))

	// Update the whole thread as being read
	dl.Expect(mock.NewExpectation(dl.UpdateMember, tID, eID, &dal.MemberUpdate{LastViewed: ptr.Time(readTime)}))

	// Find any thread items created after the last time they last viewed it
	dl.Expect(mock.NewExpectation(dl.ThreadItemIDsCreatedAfter, tID, *lView).WithReturns([]models.ThreadItemID{tiID1, tiID2}, nil))

	// Create a view record for each of those items
	dl.Expect(mock.NewExpectation(dl.CreateThreadItemViewDetails, []*models.ThreadItemViewDetails{
		{
			ThreadItemID:  tiID1,
			ActorEntityID: eID,
			ViewTime:      ptr.Time(readTime),
		},
		{
			ThreadItemID:  tiID2,
			ActorEntityID: eID,
			ViewTime:      ptr.Time(readTime),
		},
	}))

	resp, err := srv.MarkThreadAsRead(nil, &threading.MarkThreadAsReadRequest{
		ThreadID: tID.String(),
		EntityID: eID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.MarkThreadAsReadResponse{}, resp)
}

func TestMarkThreadAsReadNilLastView(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID1, err := models.NewThreadItemID()
	test.OK(t, err)
	tiID2, err := models.NewThreadItemID()
	test.OK(t, err)
	eID := "entity:1"
	lView := time.Unix(0, 0)
	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil)

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadMemberships, []models.ThreadID{tID}, eID, true).WithReturns(
		[]*models.ThreadMember{
			{
				ThreadID:   tID,
				EntityID:   eID,
				LastViewed: nil,
			},
		}, nil,
	))

	// Update the whole thread as being read
	dl.Expect(mock.NewExpectation(dl.UpdateMember, tID, eID, &dal.MemberUpdate{LastViewed: ptr.Time(readTime)}))

	// Find any thread items created after the last time they last viewed it
	dl.Expect(mock.NewExpectation(dl.ThreadItemIDsCreatedAfter, tID, lView).WithReturns(
		[]models.ThreadItemID{
			tiID1,
			tiID2,
		}, nil))

	// Create a view record for each of those items
	dl.Expect(mock.NewExpectation(dl.CreateThreadItemViewDetails, []*models.ThreadItemViewDetails{
		{
			ThreadItemID:  tiID1,
			ActorEntityID: eID,
			ViewTime:      ptr.Time(readTime),
		},
		{
			ThreadItemID:  tiID2,
			ActorEntityID: eID,
			ViewTime:      ptr.Time(readTime),
		},
	}))

	resp, err := srv.MarkThreadAsRead(nil, &threading.MarkThreadAsReadRequest{
		ThreadID: tID.String(),
		EntityID: eID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.MarkThreadAsReadResponse{}, resp)
}

func TestMarkThreadAsReadExistingMembership(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID1, err := models.NewThreadItemID()
	test.OK(t, err)
	tiID2, err := models.NewThreadItemID()
	test.OK(t, err)
	eID := "entity:1"
	lView := time.Unix(0, 0)
	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil)

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadMemberships, []models.ThreadID{tID}, eID, true))

	// Update the whole thread as being read
	dl.Expect(mock.NewExpectation(dl.UpdateMember, tID, eID, &dal.MemberUpdate{LastViewed: ptr.Time(readTime)}))

	// Find any thread items created after the last time they last viewed it
	dl.Expect(mock.NewExpectation(dl.ThreadItemIDsCreatedAfter, tID, lView).WithReturns(
		[]models.ThreadItemID{
			tiID1,
			tiID2,
		}, nil))

	// Create a view record for each of those items
	dl.Expect(mock.NewExpectation(dl.CreateThreadItemViewDetails, []*models.ThreadItemViewDetails{
		{
			ThreadItemID:  tiID1,
			ActorEntityID: eID,
			ViewTime:      ptr.Time(readTime),
		},
		{
			ThreadItemID:  tiID2,
			ActorEntityID: eID,
			ViewTime:      ptr.Time(readTime),
		},
	}))

	resp, err := srv.MarkThreadAsRead(nil, &threading.MarkThreadAsReadRequest{
		ThreadID: tID.String(),
		EntityID: eID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.MarkThreadAsReadResponse{}, resp)
}

func TestNotifyMembersOfPublishMessage(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	notificationClient := mock_notification.New(t)
	defer notificationClient.Finish()
	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID, err := models.NewThreadItemID()
	test.OK(t, err)
	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)
	publishingEntity := "publishingEntity"
	orgID := "orgID"
	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient)
	csrv := srv.(*threadsServer)

	directoryClient.Expect(mock.NewExpectation(directoryClient.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: orgID,
				Members: []*directory.Entity{
					{ID: "notify1", Type: directory.EntityType_INTERNAL},
					{ID: "notify2", Type: directory.EntityType_INTERNAL},
					{ID: publishingEntity, Type: directory.EntityType_INTERNAL},
					{ID: "doNotNotify", Type: directory.EntityType_ORGANIZATION},
					{ID: "doNotNotify2", Type: directory.EntityType_EXTERNAL},
				},
			},
		},
	}, nil))

	notificationClient.Expect(mock.NewExpectation(notificationClient.SendNotification, &notification.Notification{
		ShortMessage:     "A new message is available",
		OrganizationID:   orgID,
		SavedQueryID:     sqID.String(),
		ThreadID:         tID.String(),
		MessageID:        tiID.String(),
		EntitiesToNotify: []string{"notify1", "notify2"},
	}))

	csrv.notifyMembersOfPublishMessage(context.Background(), orgID, sqID, tID, tiID, publishingEntity)
}

func TestDeleteThread(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, directoryClient)
	eID := "entity_123"
	peID := "entity_456"

	dl.Expect(mock.NewExpectation(dl.Thread, tID).WithReturns(&models.Thread{PrimaryEntityID: peID}, nil))
	directoryClient.Expect(mock.NewExpectation(directoryClient.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: peID,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{ID: peID, Type: directory.EntityType_EXTERNAL, Status: directory.EntityStatus_ACTIVE},
		},
	}, nil))
	directoryClient.Expect(mock.NewExpectation(directoryClient.DeleteEntity, &directory.DeleteEntityRequest{
		EntityID: peID,
	}).WithReturns(&directory.DeleteEntityResponse{}, nil))
	dl.Expect(mock.NewExpectation(dl.DeleteThread, tID).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.RecordThreadEvent, tID, eID, models.ThreadEventDelete).WithReturns(nil))
	resp, err := srv.DeleteThread(nil, &threading.DeleteThreadRequest{
		ThreadID:      tID.String(),
		ActorEntityID: eID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.DeleteThreadResponse{}, resp)
}

func TestDeleteThreadNoPE(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, directoryClient)
	eID := "entity_123"

	dl.Expect(mock.NewExpectation(dl.Thread, tID).WithReturns(&models.Thread{PrimaryEntityID: ""}, nil))
	dl.Expect(mock.NewExpectation(dl.DeleteThread, tID).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.RecordThreadEvent, tID, eID, models.ThreadEventDelete).WithReturns(nil))
	resp, err := srv.DeleteThread(nil, &threading.DeleteThreadRequest{
		ThreadID:      tID.String(),
		ActorEntityID: eID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.DeleteThreadResponse{}, resp)
}

func TestDeleteThreadPEInternal(t *testing.T) {
	t.Parallel()
	dl := newMockDAL(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, directoryClient)
	eID := "entity_123"
	peID := "entity_456"

	dl.Expect(mock.NewExpectation(dl.Thread, tID).WithReturns(&models.Thread{PrimaryEntityID: peID}, nil))
	directoryClient.Expect(mock.NewExpectation(directoryClient.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: peID,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{ID: peID, Type: directory.EntityType_INTERNAL, Status: directory.EntityStatus_ACTIVE},
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.DeleteThread, tID).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.RecordThreadEvent, tID, eID, models.ThreadEventDelete).WithReturns(nil))
	resp, err := srv.DeleteThread(nil, &threading.DeleteThreadRequest{
		ThreadID:      tID.String(),
		ActorEntityID: eID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.DeleteThreadResponse{}, resp)
}

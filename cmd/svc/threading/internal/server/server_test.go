package server

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
)

func init() {
	conc.Testing = true
}

func TestCreateSavedQuery(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	eid, err := models.NewSavedQueryID()
	test.OK(t, err)
	esq := &models.SavedQuery{OrganizationID: "o1", EntityID: "e1"}
	dl.Expect(mock.NewExpectation(dl.CreateSavedQuery, esq).WithReturns(eid, nil))
	srv := NewThreadsServer(dl, nil, "arn")
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

func TestCreateThread(t *testing.T) {
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
		Title:        "foo",
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
	}
	dl.Expect(mock.NewExpectation(dl.Thread, thid).WithReturns(th2, nil))

	srv := NewThreadsServer(dl, nil, "arn")
	res, err := srv.CreateThread(nil, &threading.CreateThreadRequest{
		OrganizationID: "o1",
		FromEntityID:   "e1",
		Title:          "foo",
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
			ID:            mid.String(),
			Timestamp:     uint64(now.Unix()),
			Type:          threading.ThreadItem_MESSAGE,
			Internal:      true,
			ActorEntityID: "e1",
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:  "foo",
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
			ID:                   th2.ID.String(),
			OrganizationID:       "o1",
			PrimaryEntityID:      "e1",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   ps.Summary,
		},
	}, res)
	mock.FinishAll(dl)
}

func TestThreadItem(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(dl, nil, "arn")

	eid, err := models.NewThreadItemID()
	test.OK(t, err)
	now := time.Now()
	eti := &models.ThreadItem{
		ID:            eid,
		Type:          models.ItemTypeMessage,
		Created:       now,
		Internal:      true,
		ActorEntityID: "e2",
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
	res, err := srv.ThreadItem(nil, &threading.ThreadItemRequest{
		ItemID: eid.String(),
	})
	test.OK(t, err)
	test.Equals(t, &threading.ThreadItemResponse{
		Item: &threading.ThreadItem{
			ID:            eid.String(),
			Timestamp:     uint64(now.Unix()),
			Type:          threading.ThreadItem_MESSAGE,
			Internal:      true,
			ActorEntityID: "e2",
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
	mock.FinishAll(dl)
}

func TestQueryThreads(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(dl, nil, "arn")

	orgID := "entity:1"
	peID := "entity:2"
	tID, err := models.NewThreadID()
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
				},
				Cursor: "c2",
			},
		},
	}, res)
	mock.FinishAll(dl)
}

func TestThread(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(dl, nil, "arn")

	thID, err := models.NewThreadID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()

	dl.Expect(mock.NewExpectation(dl.Thread, thID).WithReturns(
		&models.Thread{
			ID:                   thID,
			OrganizationID:       orgID,
			PrimaryEntityID:      entID,
			LastMessageTimestamp: now,
		}, nil))
	res, err := srv.Thread(nil, &threading.ThreadRequest{
		ThreadID: thID.String(),
	})
	test.OK(t, err)
	test.Equals(t, &threading.ThreadResponse{
		Thread: &threading.Thread{
			ID:                   thID.String(),
			OrganizationID:       orgID,
			PrimaryEntityID:      entID,
			LastMessageTimestamp: uint64(now.Unix()),
		},
	}, res)
	mock.FinishAll(dl)
}

func TestSavedQuery(t *testing.T) {
	dl := newMockDAL(t)
	defer dl.Finish()
	srv := NewThreadsServer(dl, nil, "arn")

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
	mock.FinishAll(dl)
}

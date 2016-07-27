package server

import (
	"testing"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	mock_directory "github.com/sprucehealth/backend/svc/directory/mock"
	mock_media "github.com/sprucehealth/backend/svc/media/mock"
	"github.com/sprucehealth/backend/svc/notification"
	mock_notification "github.com/sprucehealth/backend/svc/notification/mock"
	"github.com/sprucehealth/backend/svc/settings"
	mock_settings "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/backend/svc/threading"
)

func init() {
	conc.Testing = true
}

func TestCreateSavedQuery(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	eid, err := models.NewSavedQueryID()
	test.OK(t, err)
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	esq := &models.SavedQuery{OrganizationID: "o1", EntityID: "e1"}
	dl.Expect(mock.NewExpectation(dl.CreateSavedQuery, esq).WithReturns(eid, nil))
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")
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
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()

	now := time.Unix(1e7, 0)

	thid, err := models.NewThreadID()
	test.OK(t, err)
	th := &models.Thread{
		OrganizationID:     "o1",
		PrimaryEntityID:    "e2",
		LastMessageSummary: "summ",
		SystemTitle:        "system title",
		Type:               models.ThreadTypeExternal,
	}
	dl.Expect(mock.NewExpectation(dl.CreateThread, th).WithReturns(thid, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, thid, "e1", (*dal.ThreadEntityUpdate)(nil)).WithReturns(nil))
	th2 := &models.Thread{
		ID:                   thid,
		OrganizationID:       "o1",
		PrimaryEntityID:      "e2",
		LastMessageTimestamp: now,
		LastMessageSummary:   "summ",
		Created:              now,
		MessageCount:         0,
		Type:                 models.ThreadTypeExternal,
	}
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thid}).WithReturns([]*models.Thread{th2}, nil))

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")
	res, err := srv.CreateEmptyThread(nil, &threading.CreateEmptyThreadRequest{
		OrganizationID:  "o1",
		FromEntityID:    "e1",
		PrimaryEntityID: "e2",
		SystemTitle:     "system title",
		Summary:         "summ",
		Type:            threading.ThreadType_EXTERNAL,
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateEmptyThreadResponse{
		Thread: &threading.Thread{
			ID:                   th2.ID.String(),
			OrganizationID:       "o1",
			PrimaryEntityID:      "e2",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   "summ",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         0,
			Type:                 threading.ThreadType_EXTERNAL,
		},
	}, res)

	// Test secure external threads
	th = &models.Thread{
		OrganizationID:     "o1",
		PrimaryEntityID:    "e2",
		LastMessageSummary: "summ",
		SystemTitle:        "system title",
		Type:               models.ThreadTypeSecureExternal,
	}
	dl.Expect(mock.NewExpectation(dl.CreateThread, th).WithReturns(thid, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateThreadMembers, thid, []string{"e2", "e1", "e1"}))
	th2 = &models.Thread{
		ID:                   thid,
		OrganizationID:       "o1",
		PrimaryEntityID:      "e2",
		LastMessageTimestamp: now,
		LastMessageSummary:   "summ",
		Created:              now,
		MessageCount:         0,
		Type:                 models.ThreadTypeSecureExternal,
	}
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thid}).WithReturns([]*models.Thread{th2}, nil))

	res, err = srv.CreateEmptyThread(nil, &threading.CreateEmptyThreadRequest{
		OrganizationID:  "o1",
		FromEntityID:    "e1",
		PrimaryEntityID: "e2",
		SystemTitle:     "system title",
		Summary:         "summ",
		MemberEntityIDs: []string{"e2", "e1"},
		Type:            threading.ThreadType_SECURE_EXTERNAL,
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateEmptyThreadResponse{
		Thread: &threading.Thread{
			ID:                   th2.ID.String(),
			OrganizationID:       "o1",
			PrimaryEntityID:      "e2",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   "summ",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         0,
			Type:                 threading.ThreadType_SECURE_EXTERNAL,
		},
	}, res)
}

func TestCreateThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()

	clk := clock.NewManaged(time.Unix(1e6, 0))
	now := clk.Now()

	thid, err := models.NewThreadID()
	test.OK(t, err)
	th := &models.Thread{OrganizationID: "o1", PrimaryEntityID: "e1", Type: models.ThreadTypeExternal, SystemTitle: "system title"}
	dl.Expect(mock.NewExpectation(dl.CreateThread, th).WithReturns(thid, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, thid, "e1", (*dal.ThreadEntityUpdate)(nil)).WithReturns(nil))

	mid, err := models.NewThreadItemID()
	test.OK(t, err)

	// Update reference timestamp for mentioned entities
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, thid, "e2", &dal.ThreadEntityUpdate{
		LastReferenced: &now,
	}).WithReturns(nil))

	ps := &dal.PostMessageRequest{
		ThreadID:     thid,
		FromEntityID: "e1",
		Internal:     true,
		Title:        "foo % woo",
		Text:         "<ref id=\"e2\" type=\"entity\">Foo</ref> bar",
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
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thid}).WithReturns([]*models.Thread{th2}, nil))

	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")
	res, err := srv.CreateThread(nil, &threading.CreateThreadRequest{
		OrganizationID: "o1",
		FromEntityID:   "e1",
		MessageTitle:   "foo % woo",
		SystemTitle:    "system title",
		Text:           "<ref id=\"e2\" type=\"Entity\">Foo</ref> bar",
		Internal:       true,
		Source: &threading.Endpoint{
			ID:      "555-555-5555",
			Channel: threading.Endpoint_SMS,
		},
		Summary: "Foo bar",
		Type:    threading.ThreadType_EXTERNAL,
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
					Title:   "foo % woo",
					Text:    "<ref id=\"e2\" type=\"entity\">Foo</ref> bar",
					Summary: "Foo bar",
					Status:  threading.Message_NORMAL,
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
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         0,
		},
	}, res)
}

func TestPostMessage(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()

	clk := clock.NewManaged(time.Unix(1e6, 0))
	now := clk.Now()

	th1id, err := models.NewThreadID()
	test.OK(t, err)
	ti1id, err := models.NewThreadItemID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{th1id}).WithReturns([]*models.Thread{
		{
			ID:              th1id,
			PrimaryEntityID: "e2",
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.LinkedThread, th1id).WithReturns((*models.Thread)(nil), false, dal.ErrNotFound))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     th1id,
		FromEntityID: "e1",
		Title:        "title",
		Text:         "<ref id=\"e2\" type=\"entity\">Foo</ref> <ref id=\"e3\" type=\"entity\">Bar</ref>",
		Summary:      "summary",
		TextRefs: []*models.Reference{
			{ID: "e2", Type: models.Reference_ENTITY},
			{ID: "e3", Type: models.Reference_ENTITY},
		},
	}).WithReturns(&models.ThreadItem{
		ID:            ti1id,
		ThreadID:      th1id,
		Created:       now,
		ActorEntityID: "e1",
		Internal:      false,
		Type:          models.ItemTypeMessage,
		Data: &models.Message{
			Title:   "title",
			Text:    "<ref id=\"e2\" type=\"entity\">Foo</ref> <ref id=\"e3\" type=\"entity\">Bar</ref>",
			Status:  models.Message_NORMAL,
			Summary: "summary",
			TextRefs: []*models.Reference{
				{ID: "e2", Type: models.Reference_ENTITY},
				{ID: "e3", Type: models.Reference_ENTITY},
			},
		},
	}, nil))

	// Update reference timestamp for mentioned entities
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, th1id, "e2", &dal.ThreadEntityUpdate{
		LastReferenced: &now,
	}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, th1id, "e3", &dal.ThreadEntityUpdate{
		LastReferenced: &now,
	}).WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{th1id}, "e1", dal.ForUpdate).WithReturns(map[string]*models.ThreadEntity(nil), nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, th1id, "e1", (*dal.ThreadEntityUpdate)(nil)).WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{th1id}).WithReturns([]*models.Thread{
		{ID: th1id,
			Created:                      now,
			MessageCount:                 1,
			OrganizationID:               "o1",
			PrimaryEntityID:              "e2",
			LastExternalMessageSummary:   "summary",
			LastExternalMessageTimestamp: now,
		},
	}, nil))

	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")
	res, err := srv.PostMessage(nil, &threading.PostMessageRequest{
		ThreadID:     th1id.String(),
		FromEntityID: "e1",
		Title:        "title",
		Text:         "<ref id=\"e2\" type=\"Entity\">Foo</ref> <ref id=\"e3\" type=\"Entity\">Bar</ref>",
		Summary:      "summary",
	})
	test.OK(t, err)
	test.Equals(t, &threading.PostMessageResponse{
		Item: &threading.ThreadItem{
			ID:             ti1id.String(),
			ThreadID:       th1id.String(),
			OrganizationID: "o1",
			ActorEntityID:  "e1",
			Internal:       false,
			Type:           threading.ThreadItem_MESSAGE,
			Timestamp:      uint64(now.Unix()),
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:   "title",
					Text:    "<ref id=\"e2\" type=\"entity\">Foo</ref> <ref id=\"e3\" type=\"entity\">Bar</ref>",
					Status:  threading.Message_NORMAL,
					Summary: "summary",
					TextRefs: []*threading.Reference{
						{ID: "e2", Type: threading.Reference_ENTITY},
						{ID: "e3", Type: threading.Reference_ENTITY},
					},
				},
			},
		},
		Thread: &threading.Thread{
			ID:                   th1id.String(),
			OrganizationID:       "o1",
			PrimaryEntityID:      "e2",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   "summary",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         1,
		},
	}, res)
}

func TestPostMessage_Linked(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	now := time.Now()

	th1id, err := models.NewThreadID()
	test.OK(t, err)
	th2id, err := models.NewThreadID()
	test.OK(t, err)
	ti1id, err := models.NewThreadItemID()
	test.OK(t, err)
	ti2id, err := models.NewThreadItemID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{th1id}).WithReturns([]*models.Thread{
		{
			ID:              th1id,
			PrimaryEntityID: "e2",
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.LinkedThread, th1id).WithReturns(&models.Thread{
		ID:              th2id,
		PrimaryEntityID: "e3",
	}, false, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     th1id,
		FromEntityID: "e1",
		Title:        "title",
		Text:         "text",
		Summary:      "summary",
	}).WithReturns(&models.ThreadItem{
		ID:            ti1id,
		ThreadID:      th1id,
		Created:       now,
		ActorEntityID: "e1",
		Internal:      false,
		Type:          models.ItemTypeMessage,
		Data: &models.Message{
			Title:   "title",
			Text:    "text",
			Status:  models.Message_NORMAL,
			Summary: "summary",
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{th1id}, "e1", dal.ForUpdate).WithReturns(map[string]*models.ThreadEntity(nil), nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, th1id, "e1", (*dal.ThreadEntityUpdate)(nil)).WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     th2id,
		FromEntityID: "e3",
		Title:        "title",
		Text:         "text",
		Summary:      "Spruce: text",
	}).WithReturns(&models.ThreadItem{
		ID:            ti2id,
		ThreadID:      th2id,
		Created:       now,
		ActorEntityID: "e3",
		Internal:      false,
		Type:          models.ItemTypeMessage,
		Data: &models.Message{
			Title:   "title",
			Text:    "text",
			Status:  models.Message_NORMAL,
			Summary: "Spruce: text",
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{th1id}).WithReturns([]*models.Thread{
		{
			ID:                           th1id,
			Created:                      now,
			MessageCount:                 1,
			OrganizationID:               "o1",
			PrimaryEntityID:              "e2",
			LastExternalMessageSummary:   "summary",
			LastExternalMessageTimestamp: now,
		},
	}, nil))

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")
	res, err := srv.PostMessage(nil, &threading.PostMessageRequest{
		ThreadID:     th1id.String(),
		FromEntityID: "e1",
		Title:        "title",
		Text:         "text",
		Summary:      "summary",
	})
	test.OK(t, err)
	test.Equals(t, &threading.PostMessageResponse{
		Item: &threading.ThreadItem{
			ID:             ti1id.String(),
			ThreadID:       th1id.String(),
			OrganizationID: "o1",
			ActorEntityID:  "e1",
			Internal:       false,
			Type:           threading.ThreadItem_MESSAGE,
			Timestamp:      uint64(now.Unix()),
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:   "title",
					Text:    "text",
					Status:  threading.Message_NORMAL,
					Summary: "summary",
				},
			},
		},
		Thread: &threading.Thread{
			ID:                   th1id.String(),
			OrganizationID:       "o1",
			PrimaryEntityID:      "e2",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   "summary",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         1,
		},
	}, res)
}

func TestPostMessage_Linked_PrependSender(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	now := time.Now()

	th1id, err := models.NewThreadID()
	test.OK(t, err)
	th2id, err := models.NewThreadID()
	test.OK(t, err)
	ti1id, err := models.NewThreadItemID()
	test.OK(t, err)
	ti2id, err := models.NewThreadItemID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{th1id}).WithReturns([]*models.Thread{
		{
			ID:              th1id,
			PrimaryEntityID: "e2",
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.LinkedThread, th1id).WithReturns(&models.Thread{
		ID:              th2id,
		PrimaryEntityID: "e3",
	}, true, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     th1id,
		FromEntityID: "e1",
		Title:        "title",
		Text:         "text",
		Summary:      "summary",
	}).WithReturns(&models.ThreadItem{
		ID:            ti1id,
		ThreadID:      th1id,
		Created:       now,
		ActorEntityID: "e1",
		Internal:      false,
		Type:          models.ItemTypeMessage,
		Data: &models.Message{
			Title:   "title",
			Text:    "text",
			Status:  models.Message_NORMAL,
			Summary: "summary",
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{th1id}, "e1", dal.ForUpdate).WithReturns(map[string]*models.ThreadEntity(nil), nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, th1id, "e1", (*dal.ThreadEntityUpdate)(nil)).WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     th2id,
		FromEntityID: "e3",
		Title:        "title",
		Text:         "text",
		Summary:      "Spruce: text",
	}).WithReturns(&models.ThreadItem{
		ID:            ti2id,
		ThreadID:      th2id,
		Created:       now,
		ActorEntityID: "e3",
		Internal:      false,
		Type:          models.ItemTypeMessage,
		Data: &models.Message{
			Title:   "title",
			Text:    "dewabi: text",
			Status:  models.Message_NORMAL,
			Summary: "Spruce: text",
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{th1id}).WithReturns([]*models.Thread{
		{
			ID:                           th1id,
			Created:                      now,
			MessageCount:                 1,
			OrganizationID:               "o1",
			PrimaryEntityID:              "e2",
			LastExternalMessageSummary:   "summary",
			LastExternalMessageTimestamp: now,
		},
	}, nil))

	dir := mock_directory.New(t)
	defer dir.Finish()

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "e1",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: "e1",
				Info: &directory.EntityInfo{
					DisplayName: "dewabi",
				},
			},
		},
	}, nil))

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, "WEBDOMAIN")
	res, err := srv.PostMessage(nil, &threading.PostMessageRequest{
		ThreadID:     th1id.String(),
		FromEntityID: "e1",
		Title:        "title",
		Text:         "text",
		Summary:      "summary",
	})
	test.OK(t, err)
	test.Equals(t, &threading.PostMessageResponse{
		Item: &threading.ThreadItem{
			ID:             ti1id.String(),
			ThreadID:       th1id.String(),
			OrganizationID: "o1",
			ActorEntityID:  "e1",
			Internal:       false,
			Type:           threading.ThreadItem_MESSAGE,
			Timestamp:      uint64(now.Unix()),
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:   "title",
					Text:    "text",
					Status:  threading.Message_NORMAL,
					Summary: "summary",
				},
			},
		},
		Thread: &threading.Thread{
			ID:                   th1id.String(),
			OrganizationID:       "o1",
			PrimaryEntityID:      "e2",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   "summary",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         1,
		},
	}, res)
}

func TestCreateLinkedThreads(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()

	now := time.Unix(1e7, 0)

	th1id, err := models.NewThreadID()
	test.OK(t, err)
	th1 := &models.Thread{
		OrganizationID:     "o1",
		PrimaryEntityID:    "e1",
		LastMessageSummary: "summ",
		Type:               models.ThreadTypeSupport,
		SystemTitle:        "sys1",
	}
	dl.Expect(mock.NewExpectation(dl.CreateThread, th1).WithReturns(th1id, nil))

	th2id, err := models.NewThreadID()
	test.OK(t, err)
	th2 := &models.Thread{
		OrganizationID:     "o2",
		PrimaryEntityID:    "e2",
		LastMessageSummary: "summ",
		Type:               models.ThreadTypeSupport,
		SystemTitle:        "sys2",
	}
	dl.Expect(mock.NewExpectation(dl.CreateThread, th2).WithReturns(th2id, nil))

	dl.Expect(mock.NewExpectation(dl.CreateThreadLink, &dal.ThreadLink{ThreadID: th1id}, &dal.ThreadLink{ThreadID: th2id, PrependSender: true}).WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     th1id,
		FromEntityID: "e1",
		Internal:     false,
		Title:        "title",
		Text:         "text",
		TextRefs:     nil,
		Attachments:  nil,
		Destinations: nil,
		Summary:      "summ",
	}).WithReturns(&models.ThreadItem{}, nil))

	dl.Expect(mock.NewExpectation(dl.PostMessage, &dal.PostMessageRequest{
		ThreadID:     th2id,
		FromEntityID: "e2",
		Internal:     false,
		Title:        "title",
		Text:         "text",
		TextRefs:     nil,
		Attachments:  nil,
		Destinations: nil,
		Summary:      "summ",
	}).WithReturns(&models.ThreadItem{}, nil))

	th1res := &models.Thread{
		ID:                   th1id,
		OrganizationID:       "o1",
		PrimaryEntityID:      "e1",
		LastMessageTimestamp: now,
		LastMessageSummary:   "summ",
		Created:              now,
		MessageCount:         0,
		Type:                 models.ThreadTypeSupport,
		SystemTitle:          "sys1",
	}
	th2res := &models.Thread{
		ID:                   th2id,
		OrganizationID:       "o2",
		PrimaryEntityID:      "e2",
		LastMessageTimestamp: now,
		LastMessageSummary:   "summ",
		Created:              now,
		MessageCount:         0,
		Type:                 models.ThreadTypeSupport,
		SystemTitle:          "sys2",
	}

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{th1id, th2id}).WithReturns([]*models.Thread{th1res, th2res}, nil))

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")
	res, err := srv.CreateLinkedThreads(nil, &threading.CreateLinkedThreadsRequest{
		Organization1ID:      "o1",
		Organization2ID:      "o2",
		PrimaryEntity1ID:     "e1",
		PrimaryEntity2ID:     "e2",
		PrependSenderThread1: false,
		PrependSenderThread2: true,
		Summary:              "summ",
		Text:                 "text",
		MessageTitle:         "title",
		Type:                 threading.ThreadType_SUPPORT,
		SystemTitle1:         "sys1",
		SystemTitle2:         "sys2",
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateLinkedThreadsResponse{
		Thread1: &threading.Thread{
			ID:                   th1id.String(),
			OrganizationID:       "o1",
			PrimaryEntityID:      "e1",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   "summ",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         0,
			Type:                 threading.ThreadType_SUPPORT,
			SystemTitle:          "sys1",
		},
		Thread2: &threading.Thread{
			ID:                   th2id.String(),
			OrganizationID:       "o2",
			PrimaryEntityID:      "e2",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   "summ",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         0,
			Type:                 threading.ThreadType_SUPPORT,
			SystemTitle:          "sys2",
		},
	}, res)
}

func TestThreadItem(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")

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
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tid}).WithReturns([]*models.Thread{{OrganizationID: "orgID"}}, nil))
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
				},
			},
		},
	}, res)
}

func TestQueryThreads(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	dm := mock_directory.New(t)
	defer dm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, "WEBDOMAIN")

	orgID := "entity:1"
	peID := "entity:2"
	tID, err := models.NewThreadID()
	test.OK(t, err)
	now := time.Now()
	created := time.Now()

	dm.Expect(mock.NewExpectation(dm.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: peID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{Type: directory.EntityType_PATIENT},
		}}, nil))

	// Adhoc query

	dl.Expect(mock.NewExpectation(dl.IterateThreads, orgID, peID, true, &dal.Iterator{
		EndCursor: "c1",
		Direction: dal.FromEnd,
		Count:     11,
	}).WithReturns(&dal.ThreadConnection{
		HasMore: true,
		Edges: []dal.ThreadEdge{
			{
				Cursor: "c2",
				Thread: &models.Thread{
					ID:                           tID,
					OrganizationID:               orgID,
					PrimaryEntityID:              peID,
					LastMessageTimestamp:         now,
					LastExternalMessageTimestamp: now,
					LastExternalMessageSummary:   "ExternalSummary",
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
					Type:         models.ThreadTypeExternal,
				},
			},
		},
	}, nil))

	res, err := srv.QueryThreads(nil, &threading.QueryThreadsRequest{
		OrganizationID: orgID,
		ViewerEntityID: peID,
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
					LastMessageSummary:   "ExternalSummary",
					LastPrimaryEntityEndpoints: []*threading.Endpoint{
						{
							Channel: threading.Endpoint_SMS,
							ID:      "+1234567890",
						},
					},
					CreatedTimestamp: uint64(created.Unix()),
					MessageCount:     32,
					Unread:           true,
					Type:             threading.ThreadType_EXTERNAL,
				},
				Cursor: "c2",
			},
		},
	}, res)
}

func TestQueryThreadsWithViewer(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	dm := mock_directory.New(t)
	defer dm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	clk := clock.NewManaged(time.Unix(1e6, 0))
	now := clk.Now()

	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dm, sm, mm, "WEBDOMAIN")

	orgID := "entity:1"
	peID := "entity:2"
	tID, err := models.NewThreadID()
	test.OK(t, err)
	tID2, err := models.NewThreadID()
	test.OK(t, err)
	tID3, err := models.NewThreadID()
	test.OK(t, err)

	dm.Expect(mock.NewExpectation(dm.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: peID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{Type: directory.EntityType_INTERNAL},
		}}, nil))

	// Adhoc query
	dl.Expect(mock.NewExpectation(dl.IterateThreads, orgID, peID, false, &dal.Iterator{
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
				ThreadEntity: &models.ThreadEntity{
					ThreadID:       tID,
					EntityID:       peID,
					LastViewed:     ptr.Time(time.Unix(1, 1)),
					LastReferenced: ptr.Time(time.Unix(10, 1)),
				},
			},
			{
				Cursor: "c3",
				Thread: &models.Thread{
					ID:                   tID,
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: now,
					Created:              time.Unix(now.Unix()-1000, 0),
					MessageCount:         32,
				},
				ThreadEntity: &models.ThreadEntity{
					ThreadID:       tID,
					EntityID:       peID,
					LastViewed:     ptr.Time(time.Unix(5, 1)),
					LastReferenced: ptr.Time(time.Unix(2, 1)),
				},
			},
			{
				Cursor: "c4",
				Thread: &models.Thread{
					ID:                   tID2,
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: time.Unix(now.Unix()-1000, 0),
					Created:              time.Unix(now.Unix()-2000, 0),
					MessageCount:         33,
				},
				ThreadEntity: &models.ThreadEntity{
					ThreadID:   tID2,
					EntityID:   peID,
					LastViewed: &now,
				},
			},
			{
				Cursor: "c5",
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
					UnreadReference:      true,
					CreatedTimestamp:     uint64(time.Unix(now.Unix()-1000, 0).Unix()),
					MessageCount:         32,
				},
				Cursor: "c2",
			},
			{
				Thread: &threading.Thread{
					ID:                   tID.String(),
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: uint64(now.Unix()),
					Unread:               true,
					UnreadReference:      false,
					CreatedTimestamp:     uint64(time.Unix(now.Unix()-1000, 0).Unix()),
					MessageCount:         32,
				},
				Cursor: "c3",
			},
			{
				Thread: &threading.Thread{
					ID:                   tID2.String(),
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: uint64(time.Unix(now.Unix()-1000, 0).Unix()),
					Unread:               false,
					UnreadReference:      false,
					CreatedTimestamp:     uint64(time.Unix(now.Unix()-2000, 0).Unix()),
					MessageCount:         33,
				},
				Cursor: "c4",
			},
			{
				Thread: &threading.Thread{
					ID:                   tID3.String(),
					OrganizationID:       orgID,
					PrimaryEntityID:      peID,
					LastMessageTimestamp: uint64(now.Unix()),
					Unread:               false,
					UnreadReference:      false,
					CreatedTimestamp:     uint64(now.Unix()),
					MessageCount:         0,
				},
				Cursor: "c5",
			},
		},
	}, res)
}

func TestThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	dm := mock_directory.New(t)
	defer dm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, "WEBDOMAIN")

	thID, err := models.NewThreadID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()
	created := time.Now()

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thID}).WithReturns(
		[]*models.Thread{
			{
				ID:                           thID,
				OrganizationID:               orgID,
				PrimaryEntityID:              entID,
				LastMessageTimestamp:         now,
				LastExternalMessageTimestamp: now,
				LastExternalMessageSummary:   "ExternalSummary",
				Created:                      created,
				MessageCount:                 32,
			},
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
			CreatedTimestamp:     uint64(created.Unix()),
			LastMessageSummary:   "ExternalSummary",
			MessageCount:         32,
			Unread:               false,
		},
	}, res)
}

func TestThreadWithViewer(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	dm := mock_directory.New(t)
	defer dm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, "WEBDOMAIN")

	thID, err := models.NewThreadID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()
	created := time.Now()

	dm.Expect(mock.NewExpectation(dm.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{Type: directory.EntityType_INTERNAL},
		}}, nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thID}).WithReturns(
		[]*models.Thread{
			{
				ID:                   thID,
				OrganizationID:       orgID,
				PrimaryEntityID:      entID,
				LastMessageTimestamp: now,
				Created:              created,
				MessageCount:         32,
			},
		}, nil))
	// Since we have a viewer associated with this query, expect the memberships to be queried to populate read status
	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{thID}, entID).WithReturns(
		map[string]*models.ThreadEntity{
			thID.String(): {
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
			CreatedTimestamp:     uint64(created.Unix()),
			MessageCount:         32,
		},
	}, res)
}

func TestThreadWithViewerNoMembership(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	dm := mock_directory.New(t)
	defer dm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, "WEBDOMAIN")

	thID, err := models.NewThreadID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()
	created := time.Now()

	dm.Expect(mock.NewExpectation(dm.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{Type: directory.EntityType_INTERNAL},
		}}, nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thID}).WithReturns(
		[]*models.Thread{
			{
				ID:                   thID,
				OrganizationID:       orgID,
				PrimaryEntityID:      entID,
				LastMessageTimestamp: now,
				Created:              created,
				MessageCount:         32,
			},
		}, nil))
	// Since we have a viewer associated with this query, expect the memberships to be queried and return none, this should be marked as unread
	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{thID}, entID))
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
			CreatedTimestamp:     uint64(created.Unix()),
			MessageCount:         32,
		},
	}, res)
}

func TestThreadWithViewerNoMessages(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	dm := mock_directory.New(t)
	defer dm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, "WEBDOMAIN")

	thID, err := models.NewThreadID()
	test.OK(t, err)
	orgID := "o1"
	entID := "e1"
	now := time.Now()
	created := time.Now()

	dm.Expect(mock.NewExpectation(dm.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{Type: directory.EntityType_INTERNAL},
		}}, nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thID}).WithReturns(
		[]*models.Thread{
			{
				ID:                   thID,
				OrganizationID:       orgID,
				PrimaryEntityID:      entID,
				LastMessageTimestamp: now,
				Created:              created,
				MessageCount:         0,
			},
		}, nil))
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
			CreatedTimestamp:     uint64(created.Unix()),
			MessageCount:         0,
		},
	}, res)
}

func TestSavedQuery(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	dm := mock_directory.New(t)
	defer dm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, "WEBDOMAIN")

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
	dl := dalmock.New(t)
	sm := mock_settings.New(t)
	defer sm.Finish()
	defer dl.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
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
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{tID}, eID, dal.ForUpdate).WithReturns(
		map[string]*models.ThreadEntity{
			tID.String(): {
				ThreadID:   tID,
				EntityID:   eID,
				LastViewed: lView,
			},
		}, nil,
	))

	// Update the whole thread as being read
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, tID, eID, &dal.ThreadEntityUpdate{LastViewed: ptr.Time(readTime)}))

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

	resp, err := srv.MarkThreadsAsRead(nil, &threading.MarkThreadsAsReadRequest{
		ThreadWatermarks: []*threading.MarkThreadsAsReadRequest_ThreadWatermark{
			{
				ThreadID: tID.String(),
			},
		},
		EntityID: eID,
		Seen:     true,
	})
	test.OK(t, err)
	test.Equals(t, &threading.MarkThreadsAsReadResponse{}, resp)
}

func TestMarkThreadsAsRead_NotSeen(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mock_settings.New(t)
	defer sm.Finish()
	defer dl.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
	tID, err := models.NewThreadID()
	test.OK(t, err)
	tID2, err := models.NewThreadID()
	test.OK(t, err)
	eID := "entity:1"
	lView := ptr.Time(time.Unix(time.Now().Unix()-1000, 0))
	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{tID}, eID, dal.ForUpdate).WithReturns(
		map[string]*models.ThreadEntity{
			tID.String(): {
				ThreadID:   tID,
				EntityID:   eID,
				LastViewed: lView,
			},
		}, nil,
	))

	// Update the whole thread as being read
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, tID, eID, &dal.ThreadEntityUpdate{LastViewed: ptr.Time(readTime)}))

	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{tID2}, eID, dal.ForUpdate).WithReturns(
		map[string]*models.ThreadEntity{
			tID2.String(): {
				ThreadID:   tID2,
				EntityID:   eID,
				LastViewed: lView,
			},
		}, nil,
	))
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, tID2, eID, &dal.ThreadEntityUpdate{LastViewed: ptr.Time(readTime)}))

	resp, err := srv.MarkThreadsAsRead(nil, &threading.MarkThreadsAsReadRequest{
		ThreadWatermarks: []*threading.MarkThreadsAsReadRequest_ThreadWatermark{
			{
				ThreadID: tID.String(),
			},
			{
				ThreadID: tID2.String(),
			},
		},
		EntityID: eID,
		Seen:     false,
	})
	test.OK(t, err)
	test.Equals(t, &threading.MarkThreadsAsReadResponse{}, resp)
}

func TestMarkThreadAsReadNilLastView(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
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
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{tID}, eID, dal.ForUpdate).WithReturns(
		map[string]*models.ThreadEntity{
			tID.String(): {
				ThreadID:   tID,
				EntityID:   eID,
				LastViewed: nil,
			},
		}, nil,
	))

	// Update the whole thread as being read
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, tID, eID, &dal.ThreadEntityUpdate{LastViewed: ptr.Time(readTime)}))

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

	resp, err := srv.MarkThreadsAsRead(nil, &threading.MarkThreadsAsReadRequest{
		ThreadWatermarks: []*threading.MarkThreadsAsReadRequest_ThreadWatermark{
			{
				ThreadID: tID.String(),
			},
		},
		EntityID: eID,
		Seen:     true,
	})
	test.OK(t, err)
	test.Equals(t, &threading.MarkThreadsAsReadResponse{}, resp)
}

func TestMarkThreadAsReadExistingMembership(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
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
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, nil, sm, mm, "WEBDOMAIN")

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{tID}, eID, dal.ForUpdate))

	// Update the whole thread as being read
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, tID, eID, &dal.ThreadEntityUpdate{LastViewed: ptr.Time(readTime)}))

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

	resp, err := srv.MarkThreadsAsRead(nil, &threading.MarkThreadsAsReadRequest{
		ThreadWatermarks: []*threading.MarkThreadsAsReadRequest_ThreadWatermark{
			{
				ThreadID: tID.String(),
			},
		},
		EntityID: eID,
		Seen:     true,
	})
	test.OK(t, err)
	test.Equals(t, &threading.MarkThreadsAsReadResponse{}, resp)
}

func expectPreviewTeamMessageContentInNotificationEnabled(sm *mock_settings.Client, organizationID string, answer bool) {
	sm.Expect(mock.NewExpectation(sm.GetValues, &settings.GetValuesRequest{
		Keys:   []*settings.ConfigKey{{Key: threading.PreviewTeamMessageContentInNotification}},
		NodeID: organizationID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type:  settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{Boolean: &settings.BooleanValue{Value: answer}},
			},
		},
	}, nil))
}

func expectPreviewPatientMessageContentInNotificationEnabled(sm *mock_settings.Client, organizationID string, answer bool) {
	sm.Expect(mock.NewExpectation(sm.GetValues, &settings.GetValuesRequest{
		Keys:   []*settings.ConfigKey{{Key: threading.PreviewPatientMessageContentInNotification}},
		NodeID: organizationID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type:  settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{Boolean: &settings.BooleanValue{Value: answer}},
			},
		},
	}, nil))
}

func expectIsAlertAllMessagesEnabled(sm *mock_settings.Client, entityID string, answer bool) {
	sm.Expect(mock.NewExpectation(sm.GetValues, &settings.GetValuesRequest{
		Keys:   []*settings.ConfigKey{{Key: threading.AlertAllMessages}},
		NodeID: entityID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type:  settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{Boolean: &settings.BooleanValue{Value: answer}},
			},
		},
	}, nil))
}

func TestNotifyMembersOfPublishMessage(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	notificationClient := mock_notification.New(t)
	defer notificationClient.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
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
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, "WEBDOMAIN")
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
					{ID: "notify1", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(1, 0).Unix())},
					{ID: "notify2", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: "notify3", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: publishingEntity, Type: directory.EntityType_INTERNAL},
					{ID: "doNotNotify", Type: directory.EntityType_ORGANIZATION},
					{ID: "doNotNotify2", Type: directory.EntityType_EXTERNAL},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{ThreadID: tID, EntityID: "notify1", LastViewed: nil, LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify2", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify3", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: ptr.Time(clk.Now())},
		{ThreadID: tID, EntityID: publishingEntity, LastViewed: nil, LastUnreadNotify: nil},
	}, nil))

	expectPreviewPatientMessageContentInNotificationEnabled(sm, orgID, false)
	expectIsAlertAllMessagesEnabled(sm, "notify1", true)

	notificationClient.Expect(mock.NewExpectation(notificationClient.SendNotification, &notification.Notification{
		ShortMessages: map[string]string{
			"notify1": "You have a new message",
			"notify2": "You have a new mention in a thread",
			"notify3": "You have a new mention in a thread",
		},
		UnreadCounts:         nil,
		OrganizationID:       orgID,
		SavedQueryID:         sqID.String(),
		ThreadID:             tID.String(),
		MessageID:            tiID.String(),
		CollapseKey:          newMessageNotificationKey,
		DedupeKey:            newMessageNotificationKey,
		EntitiesToNotify:     []string{"notify1", "notify2", "notify3"},
		EntitiesAtReferenced: map[string]struct{}{"notify2": struct{}{}, "notify3": struct{}{}},
		Type:                 notification.NewMessageOnExternalThread,
	}))

	csrv.notifyMembersOfPublishMessage(context.Background(), orgID, sqID, &models.Thread{
		ID:             tID,
		Type:           models.ThreadTypeExternal,
		OrganizationID: orgID,
	}, &models.ThreadItem{
		ID:   tiID,
		Type: models.ItemTypeMessage,
		Data: &models.Message{
			TextRefs: []*models.Reference{
				&models.Reference{
					Type: models.Reference_ENTITY,
					ID:   "notify2",
				},
				&models.Reference{
					Type: models.Reference_ENTITY,
					ID:   "notify3",
				},
			},
		},
	}, publishingEntity)
}

func TestNotifyMembersOfPublishMessage_Team(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	notificationClient := mock_notification.New(t)
	defer notificationClient.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
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
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, "WEBDOMAIN")
	csrv := srv.(*threadsServer)

	directoryClient.Expect(mock.NewExpectation(directoryClient.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"notify1", "notify3"},
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: "notify1",
			},
			{
				ID: "notify3",
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{ThreadID: tID, EntityID: "notify1", LastViewed: nil, LastUnreadNotify: nil, Member: true},
		{ThreadID: tID, EntityID: "notify2", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify3", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: ptr.Time(clk.Now()), Member: true},
		{ThreadID: tID, EntityID: publishingEntity, LastViewed: nil, LastUnreadNotify: nil},
	}, nil))

	expectPreviewTeamMessageContentInNotificationEnabled(sm, orgID, false)
	expectIsAlertAllMessagesEnabled(sm, "notify1", true)
	expectIsAlertAllMessagesEnabled(sm, "notify3", true)

	notificationClient.Expect(mock.NewExpectation(notificationClient.SendNotification, &notification.Notification{
		ShortMessages: map[string]string{
			"notify1": "You have a new message",
			"notify3": "You have a new message",
		},
		UnreadCounts:         nil,
		OrganizationID:       orgID,
		SavedQueryID:         sqID.String(),
		ThreadID:             tID.String(),
		MessageID:            tiID.String(),
		CollapseKey:          newMessageNotificationKey,
		DedupeKey:            newMessageNotificationKey,
		EntitiesToNotify:     []string{"notify1", "notify3"},
		EntitiesAtReferenced: map[string]struct{}{},
		Type:                 notification.NewMessageOnInternalThread,
	}))

	csrv.notifyMembersOfPublishMessage(context.Background(), orgID, sqID, &models.Thread{
		ID:             tID,
		Type:           models.ThreadTypeTeam,
		OrganizationID: orgID,
	}, &models.ThreadItem{
		ID:   tiID,
		Type: models.ItemTypeMessage,
		Data: &models.Message{},
	}, publishingEntity)
}

func TestNotifyMembersOfPublishMessageClearTextSupportThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	notificationClient := mock_notification.New(t)
	defer notificationClient.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
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
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, "WEBDOMAIN")
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
					{ID: "notify1", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(1, 0).Unix())},
					{ID: "notify2", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: "notify3", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: publishingEntity, Type: directory.EntityType_INTERNAL},
					{ID: "doNotNotify", Type: directory.EntityType_ORGANIZATION},
					{ID: "doNotNotify2", Type: directory.EntityType_EXTERNAL},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{ThreadID: tID, EntityID: "notify1", LastViewed: nil, LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify2", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify3", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: ptr.Time(clk.Now())},
		{ThreadID: tID, EntityID: publishingEntity, LastViewed: nil, LastUnreadNotify: nil},
	}, nil))

	expectIsAlertAllMessagesEnabled(sm, "notify1", true)
	expectIsAlertAllMessagesEnabled(sm, "notify2", true)
	expectIsAlertAllMessagesEnabled(sm, "notify3", true)

	notificationClient.Expect(mock.NewExpectation(notificationClient.SendNotification, &notification.Notification{
		ShortMessages: map[string]string{
			"notify1": "Clear Text Message",
			"notify2": "Clear Text Message",
			"notify3": "Clear Text Message",
		},
		UnreadCounts:         nil,
		OrganizationID:       orgID,
		SavedQueryID:         sqID.String(),
		ThreadID:             tID.String(),
		MessageID:            tiID.String(),
		CollapseKey:          newMessageNotificationKey,
		DedupeKey:            newMessageNotificationKey,
		EntitiesToNotify:     []string{"notify1", "notify2", "notify3"},
		EntitiesAtReferenced: map[string]struct{}{},
	}))

	csrv.notifyMembersOfPublishMessage(context.Background(), orgID, sqID, &models.Thread{
		ID:             tID,
		Type:           models.ThreadTypeSupport,
		OrganizationID: orgID,
		UserTitle:      "ThreadTitle",
	}, &models.ThreadItem{
		ID:   tiID,
		Type: models.ItemTypeMessage,
		Data: &models.Message{
			Text: "Clear Text Message",
		},
	}, publishingEntity)
}

func TestNotifyMembersOfPublishMessageClearTextEnabled(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	notificationClient := mock_notification.New(t)
	defer notificationClient.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
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
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, "WEBDOMAIN")
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
					{ID: "notify1", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(1, 0).Unix())},
					{ID: "notify2", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: "notify3", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: publishingEntity, Type: directory.EntityType_INTERNAL},
					{ID: "doNotNotify", Type: directory.EntityType_ORGANIZATION},
					{ID: "doNotNotify2", Type: directory.EntityType_EXTERNAL},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{ThreadID: tID, EntityID: "notify1", LastViewed: nil, LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify2", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify3", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: ptr.Time(clk.Now())},
		{ThreadID: tID, EntityID: publishingEntity, LastViewed: nil, LastUnreadNotify: nil},
	}, nil))

	expectPreviewPatientMessageContentInNotificationEnabled(sm, orgID, true)
	expectIsAlertAllMessagesEnabled(sm, "notify1", true)
	expectIsAlertAllMessagesEnabled(sm, "notify2", true)
	expectIsAlertAllMessagesEnabled(sm, "notify3", true)

	notificationClient.Expect(mock.NewExpectation(notificationClient.SendNotification, &notification.Notification{
		ShortMessages: map[string]string{
			"notify1": "ThreadTitle: Clear Text Message",
			"notify2": "ThreadTitle: Clear Text Message",
			"notify3": "ThreadTitle: Clear Text Message",
		},
		UnreadCounts:         nil,
		OrganizationID:       orgID,
		SavedQueryID:         sqID.String(),
		ThreadID:             tID.String(),
		MessageID:            tiID.String(),
		CollapseKey:          newMessageNotificationKey,
		DedupeKey:            newMessageNotificationKey,
		EntitiesToNotify:     []string{"notify1", "notify2", "notify3"},
		EntitiesAtReferenced: map[string]struct{}{},
		Type:                 notification.NewMessageOnExternalThread,
	}))

	csrv.notifyMembersOfPublishMessage(context.Background(), orgID, sqID, &models.Thread{
		ID:             tID,
		Type:           models.ThreadTypeExternal,
		OrganizationID: orgID,
		UserTitle:      "ThreadTitle",
	}, &models.ThreadItem{
		ID:   tiID,
		Type: models.ItemTypeMessage,
		Data: &models.Message{
			Text: "Clear Text Message",
		},
	}, publishingEntity)
}

func TestNotifyMembersOfPublishMessageSecureExternalNonInternal(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	notificationClient := mock_notification.New(t)
	defer notificationClient.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
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
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, "WEBDOMAIN")
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
					{ID: "notify1", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(1, 0).Unix())},
					{ID: "notify2", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: "notify3", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: publishingEntity, Type: directory.EntityType_INTERNAL},
					{ID: "doNotNotify", Type: directory.EntityType_ORGANIZATION},
					{ID: "doNotNotify2", Type: directory.EntityType_EXTERNAL},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{ThreadID: tID, EntityID: "notify1", LastViewed: nil, LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify2", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify3", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: ptr.Time(clk.Now())},
		{ThreadID: tID, EntityID: publishingEntity, LastViewed: nil, LastUnreadNotify: nil},
	}, nil))

	expectPreviewPatientMessageContentInNotificationEnabled(sm, orgID, false)
	expectIsAlertAllMessagesEnabled(sm, "notify1", true)
	expectIsAlertAllMessagesEnabled(sm, "notify2", true)
	expectIsAlertAllMessagesEnabled(sm, "notify3", true)
	expectIsAlertAllMessagesEnabled(sm, "patientNotify1", true)

	notificationClient.Expect(mock.NewExpectation(notificationClient.SendNotification, &notification.Notification{
		ShortMessages: map[string]string{
			"notify1":        "You have a new message",
			"notify2":        "You have a new message",
			"notify3":        "You have a new message",
			"patientNotify1": "You have a new message",
		},
		UnreadCounts:         nil,
		OrganizationID:       orgID,
		SavedQueryID:         sqID.String(),
		ThreadID:             tID.String(),
		MessageID:            tiID.String(),
		CollapseKey:          newMessageNotificationKey,
		DedupeKey:            newMessageNotificationKey,
		EntitiesToNotify:     []string{"notify1", "notify2", "notify3", "patientNotify1"},
		EntitiesAtReferenced: map[string]struct{}{},
		Type:                 notification.NewMessageOnExternalThread,
	}))

	csrv.notifyMembersOfPublishMessage(context.Background(), orgID, sqID, &models.Thread{
		ID:              tID,
		Type:            models.ThreadTypeSecureExternal,
		OrganizationID:  orgID,
		UserTitle:       "ThreadTitle",
		PrimaryEntityID: "patientNotify1",
	}, &models.ThreadItem{
		ID:   tiID,
		Type: models.ItemTypeMessage,
		Data: &models.Message{
			Text: "Clear Text Message",
		},
		Internal: false,
	}, publishingEntity)
}

func TestNotifyMembersOfPublishMessageSecureExternalInternal(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	notificationClient := mock_notification.New(t)
	defer notificationClient.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()
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
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, "WEBDOMAIN")
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
					{ID: "notify1", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(1, 0).Unix())},
					{ID: "notify2", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: "notify3", Type: directory.EntityType_INTERNAL, CreatedTimestamp: uint64(time.Unix(0, 0).Unix())},
					{ID: publishingEntity, Type: directory.EntityType_INTERNAL},
					{ID: "doNotNotify", Type: directory.EntityType_ORGANIZATION},
					{ID: "doNotNotify2", Type: directory.EntityType_EXTERNAL},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{ThreadID: tID, EntityID: "notify1", LastViewed: nil, LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify2", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: nil},
		{ThreadID: tID, EntityID: "notify3", LastViewed: ptr.Time(clk.Now()), LastUnreadNotify: ptr.Time(clk.Now())},
		{ThreadID: tID, EntityID: publishingEntity, LastViewed: nil, LastUnreadNotify: nil},
	}, nil))

	expectPreviewPatientMessageContentInNotificationEnabled(sm, orgID, false)
	expectIsAlertAllMessagesEnabled(sm, "notify1", true)
	expectIsAlertAllMessagesEnabled(sm, "notify2", true)
	expectIsAlertAllMessagesEnabled(sm, "notify3", true)

	notificationClient.Expect(mock.NewExpectation(notificationClient.SendNotification, &notification.Notification{
		ShortMessages: map[string]string{
			"notify1": "You have a new message",
			"notify2": "You have a new message",
			"notify3": "You have a new message",
		},
		UnreadCounts:         nil,
		OrganizationID:       orgID,
		SavedQueryID:         sqID.String(),
		ThreadID:             tID.String(),
		MessageID:            tiID.String(),
		CollapseKey:          newMessageNotificationKey,
		DedupeKey:            newMessageNotificationKey,
		EntitiesToNotify:     []string{"notify1", "notify2", "notify3"},
		EntitiesAtReferenced: map[string]struct{}{},
		Type:                 notification.NewMessageOnExternalThread,
	}))

	csrv.notifyMembersOfPublishMessage(context.Background(), orgID, sqID, &models.Thread{
		ID:              tID,
		Type:            models.ThreadTypeSecureExternal,
		OrganizationID:  orgID,
		UserTitle:       "ThreadTitle",
		PrimaryEntityID: "patientNotify1",
	}, &models.ThreadItem{
		ID:   tiID,
		Type: models.ItemTypeMessage,
		Data: &models.Message{
			Text: "Clear Text Message",
		},
		Internal: true,
	}, publishingEntity)
}

func TestUpdateThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, directoryClient, sm, mm, "WEBDOMAIN")

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
		{
			ID:             tID,
			OrganizationID: "org",
			Type:           models.ThreadTypeTeam,
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{EntityID: "ent1", Member: true},
		{EntityID: "ent2", Member: true},
		{EntityID: "ent3", Member: false},
	}, nil))

	directoryClient.Expect(mock.NewExpectation(directoryClient.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"ent1", "ent4"},
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{ID: "ent1", Info: &directory.EntityInfo{DisplayName: "name1"}, Memberships: []*directory.Entity{{ID: "org"}}},
			{ID: "ent4", Info: &directory.EntityInfo{DisplayName: "name4"}, Memberships: []*directory.Entity{{ID: "org"}}},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThreadMembers, tID, []string{"ent1", "ent4"}).WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThread, tID, &dal.ThreadUpdate{
		UserTitle:   ptr.String("NewUserTitle"),
		SystemTitle: ptr.String("name1, name4"),
	}).WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
		{
			ID:                   tID,
			UserTitle:            "NewUserTitle",
			SystemTitle:          "name1, name4",
			Created:              time.Unix(1, 0),
			LastMessageTimestamp: time.Unix(1, 0),
		},
	}, nil))

	resp, err := srv.UpdateThread(nil, &threading.UpdateThreadRequest{
		ThreadID:              tID.String(),
		UserTitle:             "NewUserTitle",
		AddMemberEntityIDs:    []string{"ent4"},
		RemoveMemberEntityIDs: []string{"ent2"},
	})
	test.OK(t, err)
	test.Equals(t, &threading.UpdateThreadResponse{
		Thread: &threading.Thread{
			ID:                   tID.String(),
			CreatedTimestamp:     1,
			LastMessageTimestamp: 1,
			UserTitle:            "NewUserTitle",
			SystemTitle:          "name1, name4",
		},
	}, resp)
}

func TestUpdateThread_LastPersonLeaves(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, directoryClient, sm, mm, "WEBDOMAIN")

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
		{
			ID:             tID,
			OrganizationID: "org",
			Type:           models.ThreadTypeTeam,
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{EntityID: "ent1", Member: true},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThreadMembers, tID, []string{}).WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThread, tID, &dal.ThreadUpdate{
		SystemTitle: ptr.String(""),
	}).WithReturns(nil))

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
		{
			ID:                   tID,
			SystemTitle:          "",
			Created:              time.Unix(1, 0),
			LastMessageTimestamp: time.Unix(1, 0),
		},
	}, nil))

	resp, err := srv.UpdateThread(nil, &threading.UpdateThreadRequest{
		ThreadID:              tID.String(),
		RemoveMemberEntityIDs: []string{"ent1"},
	})
	test.OK(t, err)
	test.Equals(t, &threading.UpdateThreadResponse{
		Thread: &threading.Thread{
			ID:                   tID.String(),
			CreatedTimestamp:     1,
			LastMessageTimestamp: 1,
			SystemTitle:          "",
		},
	}, resp)
}

func TestDeleteThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, directoryClient, sm, mm, "WEBDOMAIN")
	eID := "entity_123"
	peID := "entity_456"

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{{PrimaryEntityID: peID}}, nil))
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
	dl := dalmock.New(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, directoryClient, sm, mm, "WEBDOMAIN")
	eID := "entity_123"

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{{PrimaryEntityID: ""}}, nil))
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
	dl := dalmock.New(t)
	defer dl.Finish()
	directoryClient := mock_directory.New(t)
	defer directoryClient.Finish()
	sm := mock_settings.New(t)
	defer sm.Finish()
	mm := mock_media.New(t)
	defer mm.Finish()

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, directoryClient, sm, mm, "WEBDOMAIN")
	eID := "entity_123"
	peID := "entity_456"

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{{PrimaryEntityID: peID}}, nil))
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

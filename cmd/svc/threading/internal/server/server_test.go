package server

import (
	"context"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	mockdirectory "github.com/sprucehealth/backend/svc/directory/mock"
	mockmedia "github.com/sprucehealth/backend/svc/media/mock"
	"github.com/sprucehealth/backend/svc/notification"
	mocknotification "github.com/sprucehealth/backend/svc/notification/mock"
	"github.com/sprucehealth/backend/svc/settings"
	mocksettings "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/backend/svc/threading"
)

func init() {
	conc.Testing = true
}

func TestCreateSavedQuery(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	tid1, err := models.NewThreadID()
	test.OK(t, err)
	eid, err := models.NewSavedQueryID()
	test.OK(t, err)
	esq := &models.SavedQuery{
		EntityID: "e1",
		Title:    "Stuff",
		Ordinal:  2,
		Query: &models.Query{
			Expressions: []*models.Expr{
				{Not: true, Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}},
				{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_PATIENT}},
				{Value: &models.Expr_Token{Token: "tooooooke"}},
			},
		},
	}
	dl.Expect(mock.NewExpectation(dl.CreateSavedQuery, esq).WithReturns(eid, nil))

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "e1",
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_PATIENT,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "e1",
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{ID: "o1", Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.RemoveAllItemsFromSavedQueryIndex, eid))

	dl.Expect(mock.NewExpectation(dl.IterateThreads, esq.Query, []string{"e1", "o1"}, "e1", false, &dal.Iterator{Count: 5000}).WithReturns(
		&dal.ThreadConnection{
			Edges: []dal.ThreadEdge{
				{
					Thread: &models.Thread{
						ID: tid1,
					},
					ThreadEntity: &models.ThreadEntity{
						EntityID: "e1",
						ThreadID: tid1,
					},
				},
			},
			HasMore: false,
		}, nil))

	query := &threading.Query{
		Expressions: []*threading.Expr{
			{Not: true, Value: &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_UNREAD}},
			{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}},
			{Value: &threading.Expr_Token{Token: "tooooooke"}},
		},
	}
	res, err := srv.CreateSavedQuery(nil, &threading.CreateSavedQueryRequest{
		EntityID: "e1",
		Title:    "Stuff",
		Query:    query,
		Ordinal:  2,
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateSavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID:       eid.String(),
			Title:    "Stuff",
			Query:    query,
			Ordinal:  2,
			EntityID: "e1",
		},
	}, res)
}

func TestCreateEmptyThread_Team(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	now := time.Unix(1e7, 0)
	sqid1, err := models.NewSavedQueryID()
	test.OK(t, err)
	sqid2, err := models.NewSavedQueryID()
	test.OK(t, err)
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	thid, err := models.NewThreadID()
	test.OK(t, err)
	th := &models.Thread{
		OrganizationID:     "o1",
		LastMessageSummary: "summ",
		SystemTitle:        "name1, name2",
		Type:               models.ThreadTypeTeam,
	}
	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"e1", "e2"},
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
			{ID: "e1", Info: &directory.EntityInfo{DisplayName: "name1"}, Memberships: []*directory.Entity{{ID: "o1"}}},
			{ID: "e2", Info: &directory.EntityInfo{DisplayName: "name2"}, Memberships: []*directory.Entity{{ID: "o1"}}},
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.CreateThread, th).WithReturns(thid, nil))
	dl.Expect(mock.NewExpectation(dl.AddThreadMembers, thid, []string{"e1", "e2"}))
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, thid, "e1", (*dal.ThreadEntityUpdate)(nil)).WithReturns(nil))
	th2 := &models.Thread{
		ID:                   thid,
		OrganizationID:       "o1",
		LastMessageTimestamp: now,
		LastMessageSummary:   "summ",
		Created:              now,
		MessageCount:         0,
		Type:                 models.ThreadTypeTeam,
	}
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thid}).WithReturns([]*models.Thread{th2}, nil))

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"e1", "e2"},
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_ORGANIZATION,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "e1",
				Type: directory.EntityType_INTERNAL,
			},
			{
				ID:   "e2",
				Type: directory.EntityType_INTERNAL,
			},
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.SavedQueries, "e1").WithReturns([]*models.SavedQuery{{ID: sqid1, EntityID: "e1", Query: &models.Query{}}}, nil))
	dl.Expect(mock.NewExpectation(dl.SavedQueries, "e2").WithReturns([]*models.SavedQuery{
		{ID: sqid2, EntityID: "e2", Query: &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}}}}}, nil))
	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: thid, SavedQueryID: sqid1, Timestamp: now}}))

	res, err := srv.CreateEmptyThread(nil, &threading.CreateEmptyThreadRequest{
		OrganizationID:  "o1",
		FromEntityID:    "e1",
		Summary:         "summ",
		MemberEntityIDs: []string{"e1", "e2"},
		Type:            threading.THREAD_TYPE_TEAM,
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateEmptyThreadResponse{
		Thread: &threading.Thread{
			ID:                   th2.ID.String(),
			OrganizationID:       "o1",
			LastMessageTimestamp: uint64(now.Unix()),
			LastMessageSummary:   "summ",
			CreatedTimestamp:     uint64(now.Unix()),
			MessageCount:         0,
			Type:                 threading.THREAD_TYPE_TEAM,
		},
	}, res)
}

func TestCreateEmptyThread_SecureExternal(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	now := time.Unix(1e7, 0)
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	thid, err := models.NewThreadID()
	test.OK(t, err)
	sqid1, err := models.NewSavedQueryID()
	test.OK(t, err)

	// Test secure external threads
	th := &models.Thread{
		OrganizationID:     "o1",
		PrimaryEntityID:    "e2",
		LastMessageSummary: "summ",
		SystemTitle:        "system title",
		Type:               models.ThreadTypeSecureExternal,
	}
	dl.Expect(mock.NewExpectation(dl.CreateThread, th).WithReturns(thid, nil))
	dl.Expect(mock.NewExpectation(dl.AddThreadMembers, thid, []string{"o1"}))
	th2 := &models.Thread{
		ID:                   thid,
		PrimaryEntityID:      "e2",
		OrganizationID:       "o1",
		LastMessageTimestamp: now,
		LastMessageSummary:   "summ",
		Created:              now,
		MessageCount:         0,
		Type:                 models.ThreadTypeSecureExternal,
	}
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, thid, "e1", (*dal.ThreadEntityUpdate)(nil)))
	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{thid}).WithReturns([]*models.Thread{th2}, nil))

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"o1"},
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_ORGANIZATION,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "o1",
				Type: directory.EntityType_ORGANIZATION,
				Members: []*directory.Entity{
					{ID: "e1", Type: directory.EntityType_INTERNAL},
				},
			},
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.SavedQueries, "e1").WithReturns([]*models.SavedQuery{{ID: sqid1, EntityID: "e1", Query: &models.Query{}}}, nil))
	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: thid, SavedQueryID: sqid1, Timestamp: now}}))

	res, err := srv.CreateEmptyThread(nil, &threading.CreateEmptyThreadRequest{
		OrganizationID:  "o1",
		FromEntityID:    "e1",
		PrimaryEntityID: "e2",
		SystemTitle:     "system title",
		Summary:         "summ",
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
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
			Type:                 threading.THREAD_TYPE_SECURE_EXTERNAL,
		},
	}, res)
}

func TestCreateThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	clk := clock.NewManaged(time.Unix(1e6, 0))
	now := clk.Now()

	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	thid, err := models.NewThreadID()
	test.OK(t, err)
	sqid, err := models.NewSavedQueryID()
	test.OK(t, err)
	mid, err := models.NewThreadItemID()
	test.OK(t, err)

	th := &models.Thread{OrganizationID: "o1", PrimaryEntityID: "e1", Type: models.ThreadTypeExternal, SystemTitle: "system title"}
	dl.Expect(mock.NewExpectation(dl.CreateThread, th).WithReturns(thid, nil))
	dl.Expect(mock.NewExpectation(dl.AddThreadMembers, thid, []string{"o1"}))
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, thid, "e1", (*dal.ThreadEntityUpdate)(nil)).WithReturns(nil))

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
			Channel: models.ENDPOINT_CHANNEL_SMS,
		},
		TextRefs: []*models.Reference{
			{ID: "e2", Type: models.REFERENCE_TYPE_ENTITY},
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
			Status:   models.MESSAGE_STATUS_NORMAL,
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

	// Update saved query indexes

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"o1"},
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_ORGANIZATION,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "o1",
				Type: directory.EntityType_ORGANIZATION,
				Members: []*directory.Entity{
					{ID: "e1", Type: directory.EntityType_INTERNAL},
				},
			},
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.SavedQueries, "e1").WithReturns([]*models.SavedQuery{{ID: sqid, EntityID: "e1", Query: &models.Query{}}}, nil))
	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: thid, SavedQueryID: sqid, Timestamp: now}}))

	res, err := srv.CreateThread(nil, &threading.CreateThreadRequest{
		OrganizationID: "o1",
		FromEntityID:   "e1",
		MessageTitle:   "foo % woo",
		SystemTitle:    "system title",
		Text:           "<ref id=\"e2\" type=\"Entity\">Foo</ref> bar",
		Internal:       true,
		Source: &threading.Endpoint{
			ID:      "555-555-5555",
			Channel: threading.ENDPOINT_CHANNEL_SMS,
		},
		Summary: "Foo bar",
		Type:    threading.THREAD_TYPE_EXTERNAL,
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateThreadResponse{
		ThreadID: thid.String(),
		ThreadItem: &threading.ThreadItem{
			ID:             mid.String(),
			Timestamp:      uint64(now.Unix()),
			Type:           threading.THREAD_ITEM_TYPE_MESSAGE,
			Internal:       true,
			ActorEntityID:  "e1",
			ThreadID:       th2.ID.String(),
			OrganizationID: "o1",
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:   "foo % woo",
					Text:    "<ref id=\"e2\" type=\"entity\">Foo</ref> bar",
					Summary: "Foo bar",
					Status:  threading.MESSAGE_STATUS_NORMAL,
					Source: &threading.Endpoint{
						ID:      "555-555-5555",
						Channel: threading.ENDPOINT_CHANNEL_SMS,
					},
					TextRefs: []*threading.Reference{
						{ID: "e2", Type: threading.REFERENCE_TYPE_ENTITY},
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
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	clk := clock.NewManaged(time.Unix(1e6, 0))
	now := clk.Now()

	th1id, err := models.NewThreadID()
	test.OK(t, err)
	ti1id, err := models.NewThreadItemID()
	test.OK(t, err)

	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

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
			{ID: "e2", Type: models.REFERENCE_TYPE_ENTITY},
			{ID: "e3", Type: models.REFERENCE_TYPE_ENTITY},
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
			Status:  models.MESSAGE_STATUS_NORMAL,
			Summary: "summary",
			TextRefs: []*models.Reference{
				{ID: "e2", Type: models.REFERENCE_TYPE_ENTITY},
				{ID: "e3", Type: models.REFERENCE_TYPE_ENTITY},
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

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, th1id).WithReturns([]*models.ThreadEntity{}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, th1id))

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
			Type:           threading.THREAD_ITEM_TYPE_MESSAGE,
			Timestamp:      uint64(now.Unix()),
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:   "title",
					Text:    "<ref id=\"e2\" type=\"entity\">Foo</ref> <ref id=\"e3\" type=\"entity\">Bar</ref>",
					Status:  threading.MESSAGE_STATUS_NORMAL,
					Summary: "summary",
					TextRefs: []*threading.Reference{
						{ID: "e2", Type: threading.REFERENCE_TYPE_ENTITY},
						{ID: "e3", Type: threading.REFERENCE_TYPE_ENTITY},
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
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	now := time.Now()

	th1id, err := models.NewThreadID()
	test.OK(t, err)
	th2id, err := models.NewThreadID()
	test.OK(t, err)
	ti1id, err := models.NewThreadItemID()
	test.OK(t, err)
	ti2id, err := models.NewThreadItemID()
	test.OK(t, err)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

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
			Status:  models.MESSAGE_STATUS_NORMAL,
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
			Status:  models.MESSAGE_STATUS_NORMAL,
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

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, th1id).WithReturns([]*models.ThreadEntity{}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, th1id))

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
			Type:           threading.THREAD_ITEM_TYPE_MESSAGE,
			Timestamp:      uint64(now.Unix()),
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:   "title",
					Text:    "text",
					Status:  threading.MESSAGE_STATUS_NORMAL,
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
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	now := time.Now()

	th1id, err := models.NewThreadID()
	test.OK(t, err)
	th2id, err := models.NewThreadID()
	test.OK(t, err)
	ti1id, err := models.NewThreadItemID()
	test.OK(t, err)
	ti2id, err := models.NewThreadItemID()
	test.OK(t, err)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

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
			Status:  models.MESSAGE_STATUS_NORMAL,
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
			Status:  models.MESSAGE_STATUS_NORMAL,
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

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, th1id).WithReturns([]*models.ThreadEntity{}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, th1id))

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
			Type:           threading.THREAD_ITEM_TYPE_MESSAGE,
			Timestamp:      uint64(now.Unix()),
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:   "title",
					Text:    "text",
					Status:  threading.MESSAGE_STATUS_NORMAL,
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
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	now := time.Unix(1e7, 0)
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	th1id, err := models.NewThreadID()
	test.OK(t, err)
	sqid1, err := models.NewSavedQueryID()
	test.OK(t, err)
	sqid2, err := models.NewSavedQueryID()
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
	dl.Expect(mock.NewExpectation(dl.AddThreadMembers, th1id, []string{"o1"}))
	dl.Expect(mock.NewExpectation(dl.AddThreadMembers, th2id, []string{"o2"}))
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

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"o1"},
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_ORGANIZATION,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "o1",
				Type: directory.EntityType_ORGANIZATION,
				Members: []*directory.Entity{
					{ID: "e1", Type: directory.EntityType_INTERNAL},
				},
			},
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.SavedQueries, "e1").WithReturns([]*models.SavedQuery{{ID: sqid1, EntityID: "e1", Query: &models.Query{}}}, nil))
	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: th1id, SavedQueryID: sqid1, Timestamp: now}}))

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"o2"},
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_ORGANIZATION,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "o2",
				Type: directory.EntityType_ORGANIZATION,
				Members: []*directory.Entity{
					{ID: "e2", Type: directory.EntityType_INTERNAL},
				},
			},
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.SavedQueries, "e2").WithReturns([]*models.SavedQuery{{ID: sqid2, EntityID: "e2", Query: &models.Query{}}}, nil))
	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: th2id, SavedQueryID: sqid2, Timestamp: now}}))

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
		Type:                 threading.THREAD_TYPE_SUPPORT,
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
			Type:                 threading.THREAD_TYPE_SUPPORT,
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
			Type:                 threading.THREAD_TYPE_SUPPORT,
			SystemTitle:          "sys2",
		},
	}, res)
}

func TestThreadItem(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, sm, mm)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, nil, sm, mm, nil, "WEBDOMAIN")

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
			Status: models.MESSAGE_STATUS_NORMAL,
			Source: &models.Endpoint{
				ID:      "555-555-5555",
				Channel: models.ENDPOINT_CHANNEL_VOICE,
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
			Type:           threading.THREAD_ITEM_TYPE_MESSAGE,
			Internal:       true,
			ActorEntityID:  "e2",
			ThreadID:       tid.String(),
			OrganizationID: "orgID",
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Title:  "abc",
					Text:   "hello",
					Status: threading.MESSAGE_STATUS_NORMAL,
					Source: &threading.Endpoint{
						ID:      "555-555-5555",
						Channel: threading.ENDPOINT_CHANNEL_VOICE,
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
	sm := mocksettings.New(t)
	dm := mockdirectory.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, sm, dm, mm)

	clk := clock.NewManaged(time.Unix(1e6, 0))
	now := clk.Now()

	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dm, sm, mm, nil, "WEBDOMAIN")

	orgID := "entity:1"
	peID := "entity:2"
	tID, err := models.NewThreadID()
	test.OK(t, err)
	tID2, err := models.NewThreadID()
	test.OK(t, err)
	tID3, err := models.NewThreadID()
	test.OK(t, err)

	// Adhoc query

	dm.Expect(mock.NewExpectation(dm.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: peID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_PATIENT,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   peID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{ID: orgID, Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))

	query := &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_PATIENT}}}}
	dl.Expect(mock.NewExpectation(dl.IterateThreads, query, []string{peID, orgID}, peID, false, &dal.Iterator{
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
					Type:                 models.ThreadTypeExternal,
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
					Type:                 models.ThreadTypeSecureExternal,
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
					Type:                 models.ThreadTypeExternal,
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
					Type:                 models.ThreadTypeSecureExternal,
				},
			},
		},
	}, nil))

	res, err := srv.QueryThreads(nil, &threading.QueryThreadsRequest{
		ViewerEntityID: peID,
		Iterator: &threading.Iterator{
			EndCursor: "c1",
			Direction: threading.ITERATOR_DIRECTION_FROM_END,
			Count:     11,
		},
		Type: threading.QUERY_THREADS_TYPE_ADHOC,
		QueryType: &threading.QueryThreadsRequest_Query{
			Query: &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}}}},
		},
	})
	test.OK(t, err)
	test.Equals(t, &threading.QueryThreadsResponse{
		Total:     4,
		TotalType: threading.VALUE_TYPE_UNKNOWN,
		HasMore:   true,
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
					Type:                 threading.THREAD_TYPE_EXTERNAL,
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
					Type:                 threading.THREAD_TYPE_SECURE_EXTERNAL,
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
					Type:                 threading.THREAD_TYPE_EXTERNAL,
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
					Type:                 threading.THREAD_TYPE_SECURE_EXTERNAL,
				},
				Cursor: "c5",
			},
		},
	}, res)

	// Saved query

	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.SavedQuery, sqID).WithReturns(&models.SavedQuery{ID: sqID, Query: query, EntityID: peID, Total: 11, Unread: 6}, nil))

	dm.Expect(mock.NewExpectation(dm.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: peID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_PATIENT,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   peID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{ID: orgID, Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.IterateThreadsInSavedQuery, sqID, peID, &dal.Iterator{
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
					Type:                 models.ThreadTypeExternal,
				},
				ThreadEntity: &models.ThreadEntity{
					ThreadID:       tID,
					EntityID:       peID,
					LastViewed:     ptr.Time(time.Unix(1, 1)),
					LastReferenced: ptr.Time(time.Unix(10, 1)),
				},
			},
		},
	}, nil))

	res, err = srv.QueryThreads(nil, &threading.QueryThreadsRequest{
		ViewerEntityID: peID,
		Iterator: &threading.Iterator{
			EndCursor: "c1",
			Direction: threading.ITERATOR_DIRECTION_FROM_END,
			Count:     11,
		},
		Type: threading.QUERY_THREADS_TYPE_SAVED,
		QueryType: &threading.QueryThreadsRequest_SavedQueryID{
			SavedQueryID: sqID.String(),
		},
	})
	test.OK(t, err)
	test.Equals(t, &threading.QueryThreadsResponse{
		Total:     11,
		TotalType: threading.VALUE_TYPE_EXACT,
		HasMore:   true,
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
					Type:                 threading.THREAD_TYPE_EXTERNAL,
				},
				Cursor: "c2",
			},
		},
	}, res)
}

func TestThread(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	sm := mocksettings.New(t)
	defer sm.Finish()
	dm := mockdirectory.New(t)
	defer dm.Finish()
	mm := mockmedia.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, nil, "WEBDOMAIN")

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
	sm := mocksettings.New(t)
	defer sm.Finish()
	dm := mockdirectory.New(t)
	defer dm.Finish()
	mm := mockmedia.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, nil, "WEBDOMAIN")

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
	sm := mocksettings.New(t)
	defer sm.Finish()
	dm := mockdirectory.New(t)
	defer dm.Finish()
	mm := mockmedia.New(t)
	defer mm.Finish()
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, nil, "WEBDOMAIN")

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
	sm := mocksettings.New(t)
	dm := mockdirectory.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, sm, dm, mm)

	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, nil, "WEBDOMAIN")

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
	sm := mocksettings.New(t)
	dm := mockdirectory.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, sm, dm, mm)
	srv := NewThreadsServer(clock.New(), dl, nil, "arn", nil, dm, sm, mm, nil, "WEBDOMAIN")

	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)
	entID := "e1"
	now := time.Now()

	dl.Expect(mock.NewExpectation(dl.SavedQuery, sqID).WithReturns(
		&models.SavedQuery{
			ID:       sqID,
			EntityID: entID,
			Title:    "Foo",
			Unread:   1,
			Total:    9,
			Query: &models.Query{
				Expressions: []*models.Expr{
					{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD_REFERENCE}},
				},
			},
			Created:  now,
			Modified: now,
		}, nil))
	res, err := srv.SavedQuery(nil, &threading.SavedQueryRequest{
		SavedQueryID: sqID.String(),
	})
	test.OK(t, err)
	test.Equals(t, &threading.SavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID:       sqID.String(),
			Title:    "Foo",
			Unread:   1,
			Total:    9,
			EntityID: entID,
			Query: &threading.Query{
				Expressions: []*threading.Expr{
					{Value: &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_UNREAD_REFERENCE}},
				},
			},
		},
	}, res)
}

func TestMarkThreadAsRead(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(sm, dl, mm, dir)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID1, err := models.NewThreadItemID()
	test.OK(t, err)
	tiID2, err := models.NewThreadItemID()
	test.OK(t, err)
	sq1ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	sq2ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	eID := "entity:1"
	lView := ptr.Time(time.Unix(time.Now().Unix()-1000, 0))
	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: eID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_PATIENT,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   eID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{ID: "o1", Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.ThreadsWithEntity, eID, []models.ThreadID{tID}).WithReturns(
		[]*models.Thread{
			{
				ID:                   tID,
				LastMessageTimestamp: clk.Now(),
				MessageCount:         1,
			},
		}, []*models.ThreadEntity{nil}, nil))

	dl.Expect(mock.NewExpectation(dl.SavedQueries, eID).WithReturns(
		[]*models.SavedQuery{
			{
				ID:    sq1ID,
				Query: &models.Query{},
			},
			{
				ID: sq2ID,
				Query: &models.Query{
					Expressions: []*models.Expr{
						&models.Expr{Value: &models.Expr_Flag_{
							Flag: models.EXPR_FLAG_UNREAD,
						}},
					},
				},
			},
		}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns(
		[]*models.ThreadEntity{
			{EntityID: "o1", ThreadID: tID, Member: true},
		}, nil))

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{tID}, eID).WithReturns(
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

	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: tID, SavedQueryID: sq1ID, Unread: false, Timestamp: clk.Now()},
	}))
	dl.Expect(mock.NewExpectation(dl.RemoveItemsFromSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: tID, SavedQueryID: sq2ID},
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
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(sm, dl, mm, dir)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	tID2, err := models.NewThreadID()
	test.OK(t, err)
	sq1ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	sq2ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	eID := "entity:1"
	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: eID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_PATIENT,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   eID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{ID: "o1", Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.ThreadsWithEntity, eID, []models.ThreadID{tID, tID2}).WithReturns(
		[]*models.Thread{
			{
				ID:                   tID,
				LastMessageTimestamp: clk.Now(),
				MessageCount:         1,
			},
			{
				ID:                   tID2,
				LastMessageTimestamp: clk.Now(),
				MessageCount:         1,
			},
		}, []*models.ThreadEntity{nil, nil}, nil))

	dl.Expect(mock.NewExpectation(dl.SavedQueries, eID).WithReturns(
		[]*models.SavedQuery{
			{
				ID:    sq1ID,
				Query: &models.Query{},
			},
			{
				ID: sq2ID,
				Query: &models.Query{
					Expressions: []*models.Expr{
						&models.Expr{Value: &models.Expr_Flag_{
							Flag: models.EXPR_FLAG_UNREAD,
						}},
					},
				},
			},
		}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns(
		[]*models.ThreadEntity{
			{EntityID: "o1", ThreadID: tID, Member: true},
		}, nil))

	// Update the whole thread as being read
	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, tID, eID, &dal.ThreadEntityUpdate{LastViewed: ptr.Time(readTime)}))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID2).WithReturns(
		[]*models.ThreadEntity{
			{EntityID: "o1", ThreadID: tID2, Member: true},
		}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateThreadEntity, tID2, eID, &dal.ThreadEntityUpdate{LastViewed: ptr.Time(readTime)}))

	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: tID, SavedQueryID: sq1ID, Unread: false, Timestamp: clk.Now()},
		{ThreadID: tID2, SavedQueryID: sq1ID, Unread: false, Timestamp: clk.Now()},
	}))
	dl.Expect(mock.NewExpectation(dl.RemoveItemsFromSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: tID, SavedQueryID: sq2ID},
		{ThreadID: tID2, SavedQueryID: sq2ID},
	}))

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
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID1, err := models.NewThreadItemID()
	test.OK(t, err)
	tiID2, err := models.NewThreadItemID()
	test.OK(t, err)
	sq1ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	sq2ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	eID := "entity:1"
	lView := time.Unix(0, 0)
	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: eID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_PATIENT,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   eID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{ID: "o1", Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.ThreadsWithEntity, eID, []models.ThreadID{tID}).WithReturns(
		[]*models.Thread{
			{
				ID:                   tID,
				LastMessageTimestamp: clk.Now(),
				MessageCount:         1,
			},
		}, []*models.ThreadEntity{nil}, nil))

	dl.Expect(mock.NewExpectation(dl.SavedQueries, eID).WithReturns(
		[]*models.SavedQuery{
			{
				ID:    sq1ID,
				Query: &models.Query{},
			},
			{
				ID: sq2ID,
				Query: &models.Query{
					Expressions: []*models.Expr{
						&models.Expr{Value: &models.Expr_Flag_{
							Flag: models.EXPR_FLAG_UNREAD,
						}},
					},
				},
			},
		}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns(
		[]*models.ThreadEntity{
			{EntityID: "o1", ThreadID: tID, Member: true},
		}, nil))

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{tID}, eID).WithReturns(
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

	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: tID, SavedQueryID: sq1ID, Unread: false, Timestamp: clk.Now()},
	}))
	dl.Expect(mock.NewExpectation(dl.RemoveItemsFromSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: tID, SavedQueryID: sq2ID},
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
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	dir := mockdirectory.New(t)
	defer mock.FinishAll(dl, sm, mm, dir)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	sq1ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	sq2ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	tiID1, err := models.NewThreadItemID()
	test.OK(t, err)
	tiID2, err := models.NewThreadItemID()
	test.OK(t, err)
	eID := "entity:1"
	lView := time.Unix(0, 0)
	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: eID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_PATIENT,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   eID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{ID: "o1", Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.ThreadsWithEntity, eID, []models.ThreadID{tID}).WithReturns(
		[]*models.Thread{
			{
				ID:                   tID,
				LastMessageTimestamp: clk.Now(),
				MessageCount:         1,
			},
		}, []*models.ThreadEntity{nil}, nil))

	dl.Expect(mock.NewExpectation(dl.SavedQueries, eID).WithReturns(
		[]*models.SavedQuery{
			{
				ID:    sq1ID,
				Query: &models.Query{},
			},
			{
				ID: sq2ID,
				Query: &models.Query{
					Expressions: []*models.Expr{
						&models.Expr{Value: &models.Expr_Flag_{
							Flag: models.EXPR_FLAG_UNREAD,
						}},
					},
				},
			},
		}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns(
		[]*models.ThreadEntity{
			{EntityID: "o1", ThreadID: tID, Member: true},
		}, nil))

	// Lookup the membership of the viewer in the threads records
	dl.Expect(mock.NewExpectation(dl.ThreadEntities, []models.ThreadID{tID}, eID))

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

	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: tID, SavedQueryID: sq1ID, Unread: false, Timestamp: clk.Now()},
	}))
	dl.Expect(mock.NewExpectation(dl.RemoveItemsFromSavedQueryIndex, []*dal.SavedQueryThread{
		{ThreadID: tID, SavedQueryID: sq2ID},
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

func expectPreviewTeamMessageContentInNotificationEnabled(sm *mocksettings.Client, organizationID string, answer bool) {
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

func expectPreviewPatientMessageContentInNotificationEnabled(sm *mocksettings.Client, organizationID string, answer bool) {
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

func expectIsAlertAllMessagesEnabled(sm *mocksettings.Client, entityID string, answer bool) {
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
	directoryClient := mockdirectory.New(t)
	notificationClient := mocknotification.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, directoryClient, notificationClient, sm, mm)

	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, nil, "WEBDOMAIN")
	csrv := srv.(*threadsServer)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID, err := models.NewThreadItemID()
	test.OK(t, err)
	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)
	publishingEntity := "publishingEntity"
	orgID := "orgID"

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
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify1", false)

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
		EntitiesAtReferenced: map[string]struct{}{"notify2": {}, "notify3": {}},
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
				{
					Type: models.REFERENCE_TYPE_ENTITY,
					ID:   "notify2",
				},
				{
					Type: models.REFERENCE_TYPE_ENTITY,
					ID:   "notify3",
				},
			},
		},
	}, publishingEntity)
}

func TestNotifyMembersOfPublishMessage_Team(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	directoryClient := mockdirectory.New(t)
	notificationClient := mocknotification.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, directoryClient, notificationClient, sm, mm)

	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, nil, "WEBDOMAIN")
	csrv := srv.(*threadsServer)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID, err := models.NewThreadItemID()
	test.OK(t, err)
	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)
	publishingEntity := "publishingEntity"
	orgID := "orgID"

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

	expectIsAlertAllMessagesEnabled(sm, "notify1", true)
	expectPreviewTeamMessageContentInNotificationEnabled(sm, "notify1", false)
	expectIsAlertAllMessagesEnabled(sm, "notify3", true)
	expectPreviewTeamMessageContentInNotificationEnabled(sm, "notify3", false)

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
	directoryClient := mockdirectory.New(t)
	notificationClient := mocknotification.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, directoryClient, notificationClient, sm, mm)

	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, nil, "WEBDOMAIN")
	csrv := srv.(*threadsServer)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID, err := models.NewThreadItemID()
	test.OK(t, err)
	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)
	publishingEntity := "publishingEntity"
	orgID := "orgID"

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
	directoryClient := mockdirectory.New(t)
	notificationClient := mocknotification.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, directoryClient, notificationClient, sm, mm)

	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, nil, "WEBDOMAIN")
	csrv := srv.(*threadsServer)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID, err := models.NewThreadItemID()
	test.OK(t, err)
	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)
	publishingEntity := "publishingEntity"
	orgID := "orgID"

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
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify1", true)

	expectIsAlertAllMessagesEnabled(sm, "notify2", true)
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify2", true)

	expectIsAlertAllMessagesEnabled(sm, "notify3", true)
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify3", true)

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
	directoryClient := mockdirectory.New(t)
	notificationClient := mocknotification.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, directoryClient, notificationClient, sm, mm)

	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, directoryClient, sm, mm, nil, "WEBDOMAIN")
	csrv := srv.(*threadsServer)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID, err := models.NewThreadItemID()
	test.OK(t, err)
	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)
	publishingEntity := "publishingEntity"
	orgID := "orgID"

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
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify1", false)
	expectIsAlertAllMessagesEnabled(sm, "notify2", true)
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify2", false)
	expectIsAlertAllMessagesEnabled(sm, "notify3", true)
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify3", false)
	expectIsAlertAllMessagesEnabled(sm, "patientNotify1", true)
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "patientNotify1", false)

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
	dir := mockdirectory.New(t)
	notificationClient := mocknotification.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, dir, notificationClient, sm, mm)

	readTime := time.Now()
	clk := clock.NewManaged(readTime)
	srv := NewThreadsServer(clk, dl, nil, "arn", notificationClient, dir, sm, mm, nil, "WEBDOMAIN")
	csrv := srv.(*threadsServer)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	tiID, err := models.NewThreadItemID()
	test.OK(t, err)
	sqID, err := models.NewSavedQueryID()
	test.OK(t, err)
	publishingEntity := "publishingEntity"
	orgID := "orgID"

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
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
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify1", false)

	expectIsAlertAllMessagesEnabled(sm, "notify2", true)
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify2", false)

	expectIsAlertAllMessagesEnabled(sm, "notify3", true)
	expectPreviewPatientMessageContentInNotificationEnabled(sm, "notify3", false)

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
	sm := mocksettings.New(t)
	dir := mockdirectory.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, sm, dir, mm)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	sq1ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	sq2ID, err := models.NewSavedQueryID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{
		{
			ID:             tID,
			OrganizationID: "org",
			Type:           models.ThreadTypeTeam,
		},
	}, nil))

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "ent1",
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_ORGANIZATION,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{ID: "ent1", Memberships: []*directory.Entity{{ID: "org"}}},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{EntityID: "ent1", Member: true},
		{EntityID: "ent2", Member: true},
		{EntityID: "ent3", Member: false},
	}, nil))

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
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

	// Auth membership check
	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{ThreadID: tID, EntityID: "ent1", Member: true},
		{ThreadID: tID, EntityID: "ent2", Member: true},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.AddThreadMembers, tID, []string{"ent4"}).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadMembers, tID, []string{"ent2"}).WithReturns(nil))

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

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{
		{ThreadID: tID, EntityID: "ent1", Member: true},
		{ThreadID: tID, EntityID: "ent4", Member: true},
	}, nil))
	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: []string{"ent1", "ent4"},
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_ORGANIZATION,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "ent1",
				Type: directory.EntityType_INTERNAL,
			},
			{
				ID:   "ent4",
				Type: directory.EntityType_INTERNAL,
			},
		},
	}, nil))
	dl.Expect(mock.NewExpectation(dl.SavedQueries, "ent1").WithReturns(
		[]*models.SavedQuery{
			{ID: sq1ID, Query: &models.Query{}},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.SavedQueries, "ent4").WithReturns(
		[]*models.SavedQuery{
			{ID: sq2ID, Query: &models.Query{}},
		}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, tID))
	dl.Expect(mock.NewExpectation(dl.AddItemsToSavedQueryIndex,
		[]*dal.SavedQueryThread{
			{SavedQueryID: sq1ID, ThreadID: tID, Timestamp: time.Unix(1, 0)},
			{SavedQueryID: sq2ID, ThreadID: tID, Timestamp: time.Unix(1, 0)},
		}))

	resp, err := srv.UpdateThread(nil, &threading.UpdateThreadRequest{
		ActorEntityID:         "ent1",
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
	sm := mocksettings.New(t)
	dir := mockdirectory.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, sm, dir, mm)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")

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

	dl.Expect(mock.NewExpectation(dl.RemoveThreadMembers, tID, []string{"ent1"}).WithReturns(nil))

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

	dl.Expect(mock.NewExpectation(dl.EntitiesForThread, tID).WithReturns([]*models.ThreadEntity{}, nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, tID))

	resp, err := srv.UpdateThread(nil, &threading.UpdateThreadRequest{
		ActorEntityID:         "org",
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
	dir := mockdirectory.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, dir, sm, mm)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")
	eID := "entity_123"
	peID := "entity_456"

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{{ID: tID, PrimaryEntityID: peID}}, nil))
	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: peID,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{ID: peID, Type: directory.EntityType_EXTERNAL, Status: directory.EntityStatus_ACTIVE},
		},
	}, nil))
	dir.Expect(mock.NewExpectation(dir.DeleteEntity, &directory.DeleteEntityRequest{
		EntityID: peID,
	}).WithReturns(&directory.DeleteEntityResponse{}, nil))
	dl.Expect(mock.NewExpectation(dl.DeleteThread, tID).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.RecordThreadEvent, tID, eID, models.ThreadEventDelete).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, tID))
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
	dir := mockdirectory.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, dir, sm, mm)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")
	eID := "entity_123"

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{{ID: tID, PrimaryEntityID: ""}}, nil))
	dl.Expect(mock.NewExpectation(dl.DeleteThread, tID).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.RecordThreadEvent, tID, eID, models.ThreadEventDelete).WithReturns(nil))
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, tID))
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
	dir := mockdirectory.New(t)
	sm := mocksettings.New(t)
	mm := mockmedia.New(t)
	defer mock.FinishAll(dl, dir, sm, mm)

	tID, err := models.NewThreadID()
	test.OK(t, err)
	srv := NewThreadsServer(nil, dl, nil, "arn", nil, dir, sm, mm, nil, "WEBDOMAIN")
	eID := "entity_123"
	peID := "entity_456"

	dl.Expect(mock.NewExpectation(dl.Threads, []models.ThreadID{tID}).WithReturns([]*models.Thread{{ID: tID, PrimaryEntityID: peID}}, nil))
	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
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
	dl.Expect(mock.NewExpectation(dl.RemoveThreadFromAllSavedQueryIndexes, tID))
	resp, err := srv.DeleteThread(nil, &threading.DeleteThreadRequest{
		ThreadID:      tID.String(),
		ActorEntityID: eID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.DeleteThreadResponse{}, resp)
}

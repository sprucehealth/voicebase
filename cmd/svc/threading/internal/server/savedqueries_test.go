package server

import (
	"context"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	mockdirectory "github.com/sprucehealth/backend/svc/directory/mock"
	mockmedia "github.com/sprucehealth/backend/svc/media/mock"
	mocksettings "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/backend/svc/threading"
)

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
		EntityID: "entity_1",
		Title:    "Stuff",
		Ordinal:  2,
		Type:     models.SavedQueryTypeNormal,
		Query: &models.Query{
			Expressions: []*models.Expr{
				{Not: true, Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}},
				{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_PATIENT}},
				{Value: &models.Expr_Token{Token: "tooooooke"}},
			},
		},
		NotificationsEnabled: true,
	}
	dl.Expect(mock.NewExpectation(dl.CreateSavedQuery, esq).WithReturns(eid, nil))

	dir.Expect(mock.NewExpectation(dir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "entity_1",
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
				ID:   "entity_1",
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{ID: "entity_org1", Type: directory.EntityType_ORGANIZATION},
				},
			},
		},
	}, nil))

	dl.Expect(mock.NewExpectation(dl.RemoveAllItemsFromSavedQueryIndex, eid))

	dl.Expect(mock.NewExpectation(dl.IterateThreads, esq.Query, []string{"entity_1", "entity_org1"}, "entity_1", false, &dal.Iterator{Count: 5000}).WithReturns(
		&dal.ThreadConnection{
			Edges: []dal.ThreadEdge{
				{
					Thread: &models.Thread{
						ID: tid1,
					},
					ThreadEntity: &models.ThreadEntity{
						EntityID: "entity_1",
						ThreadID: tid1,
					},
				},
			},
			HasMore: false,
		}, nil))

	dl.Expect(mock.NewExpectation(dl.RebuildNotificationsSavedQuery, "entity_1"))
	dl.Expect(mock.NewExpectation(dl.UnreadNotificationsCounts, []string{"entity_1"}))

	query := &threading.Query{
		Expressions: []*threading.Expr{
			{Not: true, Value: &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_UNREAD}},
			{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}},
			{Value: &threading.Expr_Token{Token: "tooooooke"}},
		},
	}
	res, err := srv.CreateSavedQuery(context.Background(), &threading.CreateSavedQueryRequest{
		EntityID:             "entity_1",
		Title:                "Stuff",
		Query:                query,
		Ordinal:              2,
		NotificationsEnabled: true,
		Type:                 threading.SAVED_QUERY_TYPE_NORMAL,
	})
	test.OK(t, err)
	test.Equals(t, &threading.CreateSavedQueryResponse{
		SavedQuery: &threading.SavedQuery{
			ID:                   eid.String(),
			Title:                "Stuff",
			Query:                query,
			Ordinal:              2,
			EntityID:             "entity_1",
			NotificationsEnabled: true,
			Type:                 threading.SAVED_QUERY_TYPE_NORMAL,
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
	entID := "entity_1"
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
			Type:     models.SavedQueryTypeNotifications,
		}, nil))
	res, err := srv.SavedQuery(context.Background(), &threading.SavedQueryRequest{
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
			Type: threading.SAVED_QUERY_TYPE_NOTIFICATIONS,
		},
	}, res)
}

package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestThreadsSearch(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "account_1",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		{ID: "ent", Memberships: []*directory.Entity{{ID: "org", Type: directory.EntityType_ORGANIZATION}}},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.QueryThreads, &threading.QueryThreadsRequest{
		ViewerEntityID: "ent",
		Iterator: &threading.Iterator{
			Direction: threading.ITERATOR_DIRECTION_FROM_START,
			Count:     maxThreadSearchResults,
		},
		Type: threading.QUERY_THREADS_TYPE_ADHOC,
		QueryType: &threading.QueryThreadsRequest_Query{
			Query: &threading.Query{
				Expressions: []*threading.Expr{
					{Value: &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_UNREAD}},
					{Value: &threading.Expr_Token{Token: "Zulu"}},
				},
			},
		},
	}).WithReturns(&threading.QueryThreadsResponse{
		Total:     34,
		TotalType: threading.VALUE_TYPE_EXACT,
		Edges: []*threading.ThreadEdge{
			{Cursor: "c1", Thread: &threading.Thread{ID: "t1", Type: threading.THREAD_TYPE_TEAM}},
			{Cursor: "c2", Thread: &threading.Thread{ID: "t2", Type: threading.THREAD_TYPE_TEAM}},
		},
	}, nil))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		query _ {
			threadsSearch(organizationID: "org", query: "is:unread Zulu") {
				total
				totalText
				endOfResultsText
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"threadsSearch": {
				"endOfResultsText": "2 out of 34 conversations shown\nSearch to access more",
				"total": 34,
				"totalText": "34"
			}
		}
	}`, res)
}

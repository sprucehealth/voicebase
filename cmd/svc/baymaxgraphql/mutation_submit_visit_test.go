package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestSubmitVisit(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PATIENT,
	}
	ctx = gqlctx.WithAccount(ctx, acc)
	g.svc.webDomain = "test.com"

	entityID := "entity_12345"
	visitID := "visit_12345"
	orgID := "entity_123"
	visitName := "infection"
	threadID := "threadID"

	g.ra.Expect(mock.NewExpectation(g.ra.Visit, &care.GetVisitRequest{
		ID: visitID,
	}).WithReturns(&care.GetVisitResponse{
		Visit: &care.Visit{
			OrganizationID: orgID,
			Name:           visitName,
			ID:             visitID,
			EntityID:       entityID,
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.SubmitVisit, &care.SubmitVisitRequest{
		VisitID: visitID,
	}))

	g.ra.Expect(mock.NewExpectation(g.ra.ThreadsForMember, entityID, true).WithReturns([]*threading.Thread{
		{
			ID:             threadID,
			OrganizationID: orgID,
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			ID: entityID,
			Info: &directory.EntityInfo{
				DisplayName: "Joe Schmoe",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.PostMessage, &threading.PostMessageRequest{
		FromEntityID: entityID,
		ThreadID:     threadID,
		Summary:      "Joe Schmoe: Completed a visit",
		Title:        "Completed a visit: <a href=\"https://test.com/thread/threadID/visit/visit_12345\">infection</a>",
	}).WithReturns(&threading.PostMessageResponse{
		Thread: &threading.Thread{
			ID:             threadID,
			OrganizationID: orgID,
			MessageCount:   10,
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		&directory.Entity{
			ID: orgID,
			Info: &directory.EntityInfo{
				DisplayName: "ORG NAME",
			},
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($visitID: ID!) {
		submitVisit(input: {
			clientMutationId: "a1b2c3",
			visitID: $visitID,
			}) {
				clientMutationId
				success
			}
		}`, map[string]interface{}{
		"visitID": visitID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"submitVisit": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))

}

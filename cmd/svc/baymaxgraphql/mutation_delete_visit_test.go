package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
)

func TestDeleteVisit(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PATIENT,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	visitID := "v1"
	orgID := "o1"
	entID := "e1"

	g.ra.Expect(mock.NewExpectation(g.ra.Visit, &care.GetVisitRequest{
		ID: "v1",
	}).WithReturns(&care.GetVisitResponse{
		Visit: &care.Visit{
			OrganizationID: orgID,
		},
	}, nil))

	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(
		[]*directory.Entity{
			{
				ID:   entID,
				Type: directory.EntityType_INTERNAL,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
				Memberships: []*directory.Entity{
					{ID: orgID, Type: directory.EntityType_ORGANIZATION},
				},
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.DeleteVisit, &care.DeleteVisitRequest{
		ID:            "v1",
		ActorEntityID: "e1",
	}).WithReturns(&care.DeleteVisitResponse{}, nil))

	res := g.query(ctx, `
		mutation _ ($visitID: ID!) {
			deleteVisit(input: {
				clientMutationId: "a1b2c3",
				visitID: $visitID,
			}) {
				clientMutationId
			}
		}`, map[string]interface{}{
		"visitID": visitID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"deleteVisit": {
			"clientMutationId": "a1b2c3"
		}
	}
}`, string(b))
}

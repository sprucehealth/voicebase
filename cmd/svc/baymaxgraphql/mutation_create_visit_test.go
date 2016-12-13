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
	"github.com/sprucehealth/backend/svc/layout"
)

func TestCreateVisit(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_12345",
		Type: auth.AccountType_PATIENT,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	visitLayoutID := "visitLayout_ID1"
	visitLayoutVersionID := "visitLayoutVersion_ID1"
	orgID := "o1"
	entID := "e1"

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

	g.ra.Expect(mock.NewExpectation(g.ra.VisitLayout, &layout.GetVisitLayoutRequest{
		ID: visitLayoutID,
	}).WithReturns(&layout.GetVisitLayoutResponse{
		VisitLayout: &layout.VisitLayout{
			Name: "test",
			Version: &layout.VisitLayoutVersion{
				ID:                   visitLayoutVersionID,
				IntakeLayoutLocation: "testlocation",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateVisit, &care.CreateVisitRequest{
		CreatorID:        entID,
		EntityID:         entID,
		PatientInitiated: true,
		OrganizationID:   orgID,
		LayoutVersionID:  visitLayoutVersionID,
		Name:             "test",
	}).WithReturns(&care.CreateVisitResponse{Visit: &care.Visit{ID: "vID", Preferences: &care.Visit_Preference{}}}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgID,
		},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_ORGANIZATION,
			ID:   orgID,
			Info: &directory.EntityInfo{
				DisplayName: "displayName",
			},
		},
	}, nil))

	g.layoutStore.Expect(mock.NewExpectation(g.layoutStore.GetIntake, "testlocation").WithReturns(&layout.Intake{}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.GetAnswersForVisit, &care.GetAnswersForVisitRequest{
		VisitID:              "vID",
		SerializedForPatient: true,
	}).WithReturns(&care.GetAnswersForVisitResponse{}, nil))

	res := g.query(ctx, `
		mutation _ ($visitLayoutID: ID!, $organizationID: ID!) {
			createVisit(input: {
				clientMutationId: "a1b2c3",
				visitLayoutID: $visitLayoutID,
				organizationID: $organizationID,
			}) {
				clientMutationId
			}
		}`, map[string]interface{}{
		"visitLayoutID":  visitLayoutID,
		"organizationID": orgID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"createVisit": {
			"clientMutationId": "a1b2c3"
		}
	}
}`, string(b))
}

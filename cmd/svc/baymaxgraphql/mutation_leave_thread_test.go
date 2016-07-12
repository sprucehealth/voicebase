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

func TestLeaveThreadMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "a_1",
	}
	organizationID := "e_org"
	ctx = gqlctx.WithAccount(ctx, acc)

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, "t_1", "").WithReturns(&threading.Thread{
		ID:             "t_1",
		Type:           threading.ThreadType_TEAM,
		OrganizationID: organizationID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "a_1",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		{
			ID:   "eor",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{
					ID:   organizationID,
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ThreadID:              "t_1",
		RemoveMemberEntityIDs: []string{"eor"},
	}).WithReturns(&threading.UpdateThreadResponse{
		Thread: &threading.Thread{
			ID:          "t_1",
			Type:        threading.ThreadType_TEAM,
			SystemTitle: "Person1, Person2",
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			leaveThread(input: {
				clientMutationId: "a1b2c3",
				threadID: "t_1",
			}) {
				clientMutationId
				success
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"leaveThread": {
				"clientMutationId": "a1b2c3",
				"success": true
			}
		}}`, res)
}

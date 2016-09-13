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

func TestUpdateThreadMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "a_1",
		Type: auth.AccountType_PROVIDER,
	}
	organizationID := "e_org"
	ctx = gqlctx.WithAccount(ctx, acc)

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, "t_1", "").WithReturns(&threading.Thread{
		ID:             "t_1",
		Type:           threading.THREAD_TYPE_TEAM,
		OrganizationID: organizationID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
	}).WithReturns([]*directory.Entity{
		{
			ID:   "ent",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{ID: organizationID, Type: directory.EntityType_ORGANIZATION},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ActorEntityID:         "ent",
		ThreadID:              "t_1",
		UserTitle:             "newTitle",
		AddMemberEntityIDs:    []string{"e1", "e2"},
		RemoveMemberEntityIDs: []string{"e3"},
	}).WithReturns(&threading.UpdateThreadResponse{
		Thread: &threading.Thread{
			ID:          "t_1",
			Type:        threading.THREAD_TYPE_TEAM,
			UserTitle:   "newTitle",
			SystemTitle: "Person1, Person2",
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			updateThread(input: {
				clientMutationId: "a1b2c3",
				threadID: "t_1",
				title: "newTitle",
				addMemberEntityIDs: ["e1", "e2"],
				removeMemberEntityIDs: ["e3"],
			}) {
				clientMutationId
				success
				thread {
					id
					allowInternalMessages
					allowDelete
					allowAddMembers
					allowRemoveMembers
					allowLeave
					allowUpdateTitle
					title
				}
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"updateThread": {
				"clientMutationId": "a1b2c3",
				"success": true,
				"thread": {
					"allowAddMembers": true,
					"allowDelete": false,
					"allowInternalMessages": false,
					"allowLeave": true,
					"allowRemoveMembers": true,
					"allowUpdateTitle": true,
					"id": "t_1",
					"title": "newTitle"
				}
			}
		}}`, res)
}

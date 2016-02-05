package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestDeleteThread(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account_12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	threadID := "t1"
	orgID := "o1"
	entID := "e1"

	// Fetch thread
	g.thC.Expect(mock.NewExpectation(g.thC.Thread, &threading.ThreadRequest{
		ThreadID: threadID,
	}).WithReturns(&threading.ThreadResponse{
		Thread: &threading.Thread{
			ID:             threadID,
			OrganizationID: orgID,
		},
	}, nil))

	// Looking up the account's entity for the org
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
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
		},
	}, nil))

	// Delete thread
	g.thC.Expect(mock.NewExpectation(g.thC.DeleteThread, &threading.DeleteThreadRequest{
		ThreadID:      threadID,
		ActorEntityID: entID,
	}).WithReturns(&threading.DeleteThreadResponse{}, nil))

	res := g.query(ctx, `
		mutation _ ($threadID: ID!) {
			deleteThread(input: {
				clientMutationId: "a1b2c3",
				threadID: $threadID,
			}) {
				clientMutationId
			}
		}`, map[string]interface{}{
		"threadID": threadID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"deleteThread": {
			"clientMutationId": "a1b2c3"
		}
	}
}`, string(b))
}

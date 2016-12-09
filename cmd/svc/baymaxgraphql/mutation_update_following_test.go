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

func TestUpdateFollowingForThreadsMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "a_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
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
				{ID: "org", Type: directory.EntityType_ORGANIZATION},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ActorEntityID:        "ent",
		ThreadID:             "thread1",
		AddFollowerEntityIDs: []string{"ent"},
	}).WithReturns(&threading.UpdateThreadResponse{
		Thread: &threading.Thread{
			ID:   "thread1",
			Type: threading.THREAD_TYPE_TEAM,
		},
	}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ActorEntityID:        "ent",
		ThreadID:             "thread2",
		AddFollowerEntityIDs: []string{"ent"},
	}).WithReturns(&threading.UpdateThreadResponse{
		Thread: &threading.Thread{
			ID:   "thread2",
			Type: threading.THREAD_TYPE_EXTERNAL,
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "org",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			ID:   "org",
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "OGRES",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_CONTACTS,
			},
		},
		RootTypes: []directory.EntityType{
			directory.EntityType_INTERNAL,
			directory.EntityType_PATIENT,
		},
		ChildTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			ID:   "ent",
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{},
			Memberships: []*directory.Entity{
				{ID: "org", Type: directory.EntityType_ORGANIZATION},
			},
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ {
			updateFollowingForThreads(input: {
				clientMutationId: "a1b2c3",
				orgID: "org",
				threadIDs: ["thread1", "thread2"],
				following: true
			}) {
				clientMutationId
				success
				threads {
					id
				}
				organization {
					id
				}
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"updateFollowingForThreads": {
				"clientMutationId": "a1b2c3",
				"success": true,
				"threads": [{"id": "thread1"}, {"id": "thread2"}],
				"organization": {
					"id": "org"
				}
			}
		}}`, res)
}

package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestDeleteThread(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	threadID := "t1"
	orgID := "o1"
	entID := "e1"

	// Fetch thread
	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		ID:             threadID,
		OrganizationID: orgID,
	}, nil))

	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
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

	// Delete thread
	g.ra.Expect(mock.NewExpectation(g.ra.DeleteThread, threadID, entID).WithReturns(nil))

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

func TestDeleteThread_Secure(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	threadID := "t1"
	orgID := "o1"
	entID := "e1"
	patientEntID := "e2"

	// Fetch thread
	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		ID:              threadID,
		OrganizationID:  orgID,
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		PrimaryEntityID: patientEntID,
	}, nil))

	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
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

	// look up the patient associated with the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: patientEntID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	},
	).WithReturns([]*directory.Entity{
		{
			ID:   patientEntID,
			Type: directory.EntityType_PATIENT,
		},
	}, nil))

	gomock.InOrder(
		// delete invite
		g.inviteC.EXPECT().DeleteInvite(ctx, &invite.DeleteInviteRequest{
			DeleteInviteKey: invite.DeleteInviteRequest_PARKED_ENTITY_ID,
			Key: &invite.DeleteInviteRequest_ParkedEntityID{
				ParkedEntityID: patientEntID,
			},
		}),
	)

	// Delete thread
	g.ra.Expect(mock.NewExpectation(g.ra.DeleteThread, threadID, entID).WithReturns(nil))

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

func TestDeleteThread_Secure_AccountCreated(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	threadID := "t1"
	orgID := "o1"
	entID := "e1"
	patientEntID := "e2"

	// Fetch thread
	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		ID:              threadID,
		OrganizationID:  orgID,
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		PrimaryEntityID: patientEntID,
	}, nil))

	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
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

	// look up the patient associated with the thread
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: patientEntID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	},
	).WithReturns([]*directory.Entity{
		{
			ID:        patientEntID,
			Type:      directory.EntityType_PATIENT,
			AccountID: "accountCreated",
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($threadID: ID!) {
			deleteThread(input: {
				clientMutationId: "a1b2c3",
				threadID: $threadID,
			}) {
				clientMutationId
				success
				errorCode
			}
		}`, map[string]interface{}{
		"threadID": threadID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"deleteThread": {
			"clientMutationId": "a1b2c3",
			"errorCode": "PATIENT_ALREADY_CREATED_ACCOUNT",
			"success": false
		}
	}
}`, string(b))
}

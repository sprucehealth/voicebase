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

func TestPostEventMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "a_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	// Non setup events are currently ignored
	res := g.query(ctx, `
		mutation _ {
			postEvent(input: {
				clientMutationId: "a1b2c3",
				eventName: "someEvent",
			}) {
				clientMutationId
				success
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"postEvent": {
				"clientMutationId": "a1b2c3",
				"success": true
			}
		}}`, res)

	// no org_id
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		{
			ID:   "ent",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{ID: "org", Type: directory.EntityType_ORGANIZATION},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.OnboardingThreadEvent,
		&threading.OnboardingThreadEventRequest{
			LookupByType: threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID,
			LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
				EntityID: "org",
			},
			EventType: threading.ONBOARDING_THREAD_EVENT_TYPE_GENERIC_SETUP,
			Event: &threading.OnboardingThreadEventRequest_GenericSetup{
				GenericSetup: &threading.GenericSetupEvent{
					Name:       "setup_answering_service",
					Attributes: []*threading.KeyValue{},
				},
			},
		},
	))

	res = g.query(ctx, `
		mutation _ {
			postEvent(input: {
				clientMutationId: "a1b2c3",
				eventName: "setup_answering_service",
			}) {
				clientMutationId
				success
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"postEvent": {
				"clientMutationId": "a1b2c3",
				"success": true
			}
		}}`, res)

	// with org_id

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   "ent",
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{
					ID:   "0rg",
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.OnboardingThreadEvent,
		&threading.OnboardingThreadEventRequest{
			LookupByType: threading.ONBOARDING_THREAD_LOOKUP_BY_ENTITY_ID,
			LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
				EntityID: "0rg",
			},
			EventType: threading.ONBOARDING_THREAD_EVENT_TYPE_GENERIC_SETUP,
			Event: &threading.OnboardingThreadEventRequest_GenericSetup{
				GenericSetup: &threading.GenericSetupEvent{
					Name: "setup_answering_service",
					Attributes: []*threading.KeyValue{
						{Key: "org_id", Value: "0rg"},
					},
				},
			},
		},
	))

	res = g.query(ctx, `
		mutation _ {
			postEvent(input: {
				clientMutationId: "a1b2c3",
				eventName: "setup_answering_service",
				attributes: [
					{
						key: "org_id",
						value: "0rg",
					}
				]
			}) {
				clientMutationId
				success
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"postEvent": {
				"clientMutationId": "a1b2c3",
				"success": true
			}
		}}`, res)
}

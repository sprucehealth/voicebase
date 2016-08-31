package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/payments"
)

func TestEHRLinkQuery(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	id := "entity_12345"

	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: id,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_EXTERNAL,
			ID:   id,
			Info: &directory.EntityInfo{
				DisplayName: "Someone",
				Gender:      directory.EntityInfo_MALE,
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.LookupEHRLinksForEntity, &directory.LookupEHRLinksForEntityRequest{
		EntityID: id,
	}).WithReturns(&directory.LookupEHRLinksforEntityResponse{
		Links: []*directory.LookupEHRLinksforEntityResponse_EHRLink{
			{
				Name: "drchrono",
				URL:  "https://www.drcrhono.com",
			},
			{
				Name: "hint",
				URL:  "https://www.hint.com",
			},
		},
	}, nil))

	res := g.query(ctx, `
 query _ {
   node(id: "entity_12345") {
   	__typename
   	... on Entity {
	    ehrLinks {
	    	name
	    	url
	      }
	    }   		
   	}
 }
`, nil)

	responseEquals(t, `{"data":{"node":{"__typename":"Entity","ehrLinks":[{"name":"drchrono","url":"https://www.drcrhono.com"},{"name":"hint","url":"https://www.hint.com"}]}}}`, res)
}

func TestPartnerIntegrations(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	id := "entity_12345"
	entityID := "ent_id"

	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: id,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_ORGANIZATION,
			ID:   id,
			Info: &directory.EntityInfo{
				DisplayName: "SUP",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   entityID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Memberships: []*directory.Entity{
				{
					ID:   id,
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.VendorAccounts, &payments.VendorAccountsRequest{
		EntityID: id,
	}).WithReturns(&payments.VendorAccountsResponse{VendorAccounts: []*payments.VendorAccount{{}}}, nil))

	res := g.query(ctx, `
 query _ {
   node(id: "entity_12345") {
   	__typename
   	... on Organization {
	    partnerIntegrations {
	    	connected
	    	errored
	    	title
	    	subtitle
	    	buttonText
	    	buttonURL
	      }
	    }   		
   	}
 }
`, nil)

	responseEquals(t, `{"data":{"node":{"__typename":"Organization","partnerIntegrations":[{"buttonText":"Stripe Dashboard","buttonURL":"https://dashboard.stripe.com","connected":true,"errored":false,"subtitle":"View and manage your transaction history through Stripe.","title":"Connected to Stripe"}]}}}`, res)
}

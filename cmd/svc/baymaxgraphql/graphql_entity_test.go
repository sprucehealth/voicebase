package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/payments"
)

func TestExternalLinkQuery(t *testing.T) {
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

	g.ra.Expect(mock.NewExpectation(g.ra.LookupExternalLinksForEntity, &directory.LookupExternalLinksForEntityRequest{
		EntityID: id,
	}).WithReturns(&directory.LookupExternalLinksforEntityResponse{
		Links: []*directory.LookupExternalLinksforEntityResponse_ExternalLink{
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
	    externalLinks {
	    	name
	    	url
	      }
	    }   		
   	}
 }
`, nil)

	responseEquals(t, `{"data":{"node":{"__typename":"Entity","externalLinks":[{"name":"drchrono","url":"https://www.drcrhono.com"},{"name":"hint","url":"https://www.hint.com"}]}}}`, res)
}

func TestInvitationBannerQuery_Paitent(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	id := "entity_12345"

	t.Run("Patient", func(t *testing.T) {
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
				Type: directory.EntityType_PATIENT,
				ID:   id,
				Info: &directory.EntityInfo{
					DisplayName: "Someone",
					Gender:      directory.EntityInfo_MALE,
				},
			},
		}, nil))

		g.inviteC.Expect(mock.NewExpectation(g.inviteC.LookupInvites, &invite.LookupInvitesRequest{
			LookupKeyType: invite.LookupInvitesRequest_PARKED_ENTITY_ID,
			Key: &invite.LookupInvitesRequest_ParkedEntityID{
				ParkedEntityID: id,
			},
		}).WithReturns(&invite.LookupInvitesResponse{
			List: &invite.LookupInvitesResponse_PatientInviteList{
				PatientInviteList: &invite.PatientInviteList{
					PatientInvites: []*invite.PatientInvite{
						{},
					},
				},
			},
		}, nil))

		res := g.query(ctx, `
 query _ {
   node(id: "entity_12345") {
   	__typename
   	... on Entity {
	    invitationBanner {
	    	hasPendingInvite
	      }
	    }   		
   	}
 }
`, nil)

		responseEquals(t, `{"data":{"node":{"__typename":"Entity","invitationBanner":{"hasPendingInvite":true}}}}`, res)
	})

	t.Run("External", func(t *testing.T) {
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

		res := g.query(ctx, `
 query _ {
   node(id: "entity_12345") {
   	__typename
   	... on Entity {
	    invitationBanner {
	    	hasPendingInvite
	      }
	    }
   	}
 }
`, nil)

		responseEquals(t, `{"data":{"node":{"__typename":"Entity","invitationBanner":null}}}`, res)
	})

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

	g.ra.Expect(mock.NewExpectation(g.ra.LookupPatientSyncConfiguration, &patientsync.LookupSyncConfigurationRequest{
		OrganizationEntityID: id,
		Source:               patientsync.SOURCE_HINT,
	}).WithReturns(&patientsync.LookupSyncConfigurationResponse{}, nil))

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

	responseEquals(t, `{"data":{"node":{"__typename":"Organization","partnerIntegrations":[{"buttonText":"Stripe Dashboard","buttonURL":"https://dashboard.stripe.com","connected":true,"errored":false,"subtitle":"View and manage your transaction history through Stripe.","title":"Connected to Stripe"},{"buttonText":"Hint Dashboard","buttonURL":"","connected":true,"errored":false,"subtitle":"View and manage patient membership information in Hint.","title":"Connected"}]}}}`, res)
}

package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
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

package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	ramock "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess/mock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
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

type testHasConnectedStripeAccountParams struct {
	p  graphql.ResolveParams
	rm *ramock.ResourceAccessor
}

func (t *testHasConnectedStripeAccountParams) Finishers() []mock.Finisher {
	return []mock.Finisher{t.rm}
}

func TestHasConnectedStripeAccount(t *testing.T) {
	entID := "entID"
	cases := map[string]struct {
		tp          *testHasConnectedStripeAccountParams
		Expected    interface{}
		ExpectedErr error
	}{
		"Success-True-ConnectedAccount": {
			tp: func() *testHasConnectedStripeAccountParams {
				rm := ramock.New(t)
				rm.Expect(mock.NewExpectation(rm.VendorAccounts, &payments.VendorAccountsRequest{
					EntityID: entID,
				}).WithReturns(&payments.VendorAccountsResponse{VendorAccounts: []*payments.VendorAccount{&payments.VendorAccount{}}}, nil))
				return &testHasConnectedStripeAccountParams{
					p: graphql.ResolveParams{
						Context: context.Background(),
						Source: &models.Entity{
							ID: entID,
						},
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: rm,
							},
						},
					},
					rm: rm,
				}
			}(),
			Expected:    true,
			ExpectedErr: nil,
		},
		"Success-False-NoConnectedAccount": {
			tp: func() *testHasConnectedStripeAccountParams {
				rm := ramock.New(t)
				rm.Expect(mock.NewExpectation(rm.VendorAccounts, &payments.VendorAccountsRequest{
					EntityID: entID,
				}).WithReturns(&payments.VendorAccountsResponse{VendorAccounts: []*payments.VendorAccount{}}, nil))
				return &testHasConnectedStripeAccountParams{
					p: graphql.ResolveParams{
						Context: context.Background(),
						Source: &models.Entity{
							ID: entID,
						},
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: rm,
							},
						},
					},
					rm: rm,
				}
			}(),
			Expected:    false,
			ExpectedErr: nil,
		},
	}

	for cn, c := range cases {
		out, err := resolveHasConnectedStripeAccount(c.tp.p)
		test.EqualsCase(t, cn, c.Expected, out)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		mock.FinishAll(c.tp.Finishers()...)
	}
}

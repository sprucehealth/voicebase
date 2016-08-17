package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	ramock "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess/mock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
)

type testConnectVendorAccountParams struct {
	p         graphql.ResolveParams
	finishers []mock.Finisher
}

func TestConnectVendorStripeAccount(t *testing.T) {
	mutationID := "mutationID"
	entityID := "entityID"
	code := "code"
	cases := map[string]struct {
		In          connectVendorStripeAccountInput
		TestParams  *testConnectVendorAccountParams
		Expected    interface{}
		ExpectedErr error
	}{
		"Success": {
			In: connectVendorStripeAccountInput{
				ClientMutationID: mutationID,
				EntityID:         entityID,
				Code:             code,
			},
			TestParams: func() *testConnectVendorAccountParams {
				mra := ramock.New(t)
				mra.Expect(mock.NewExpectation(mra.ConnectVendorAccount, &payments.ConnectVendorAccountRequest{
					EntityID: entityID,
					Type:     payments.VENDOR_ACCOUNT_TYPE_STRIPE,
					ConnectVendorAccountOneof: &payments.ConnectVendorAccountRequest_StripeRequest{
						StripeRequest: &payments.StripeAccountConnectRequest{
							Code: code,
						},
					},
				}))
				return &testConnectVendorAccountParams{
					p: graphql.ResolveParams{
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: mra,
							},
						},
					},
					finishers: []mock.Finisher{mra},
				}
			}(),
			ExpectedErr: nil,
			Expected: &connectVendorStripeAccountOutput{
				ClientMutationID: mutationID,
				Success:          true,
			},
		},
	}
	for cn, c := range cases {
		out, err := connectVendorStripeAccount(c.TestParams.p, c.In)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, out)
		mock.FinishAll(c.TestParams.finishers...)
	}
}

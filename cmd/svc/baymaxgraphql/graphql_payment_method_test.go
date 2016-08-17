package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	ramock "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess/mock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
)

type testPaymentMethodParams struct {
	p         graphql.ResolveParams
	finishers []mock.Finisher
}

func TestPaymentMethodEntityResolve(t *testing.T) {
	entityID := "entityID"
	cases := map[string]struct {
		TestParams  *testPaymentMethodParams
		Expected    interface{}
		ExpectedErr error
	}{
		"Success": {
			TestParams: func() *testPaymentMethodParams {
				mra := ramock.New(t)
				mra.Expect(mock.NewExpectation(mra.PaymentMethods, &payments.PaymentMethodsRequest{
					EntityID: entityID,
				}).WithReturns(&payments.PaymentMethodsResponse{
					[]*payments.PaymentMethod{
						{
							ID:          "ID",
							EntityID:    entityID,
							Default:     true,
							Lifecycle:   payments.PAYMENT_METHOD_LIFECYCLE_ACTIVE,
							ChangeState: payments.PAYMENT_METHOD_CHANGE_STATE_NONE,
							StorageType: payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE,
							Type:        payments.PAYMENT_METHOD_TYPE_CARD,
							PaymentMethodOneof: &payments.PaymentMethod_StripeCard{
								StripeCard: &payments.StripeCard{
									ID:                 "cardID",
									TokenizationMethod: "TokenizationMethod",
									Brand:              "Brand",
									Last4:              "LastFour",
								},
							},
						},
					},
				}, nil))
				return &testPaymentMethodParams{
					p: graphql.ResolveParams{
						Source: &models.Entity{ID: entityID},
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
			Expected: transformPaymentMethodsToResponse([]*payments.PaymentMethod{
				{
					ID:          "ID",
					EntityID:    entityID,
					Default:     true,
					Lifecycle:   payments.PAYMENT_METHOD_LIFECYCLE_ACTIVE,
					ChangeState: payments.PAYMENT_METHOD_CHANGE_STATE_NONE,
					StorageType: payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE,
					Type:        payments.PAYMENT_METHOD_TYPE_CARD,
					PaymentMethodOneof: &payments.PaymentMethod_StripeCard{
						StripeCard: &payments.StripeCard{
							ID:                 "cardID",
							TokenizationMethod: "TokenizationMethod",
							Brand:              "Brand",
							Last4:              "LastFour",
						},
					},
				},
			}),
		},
	}
	for cn, c := range cases {
		out, err := resolveEntityPaymentMethods(c.TestParams.p)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, out)
		mock.FinishAll(c.TestParams.finishers...)
	}
}

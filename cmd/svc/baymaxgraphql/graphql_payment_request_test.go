package main

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	ramock "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess/mock"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/payments"
)

type testPaymentRequestParams struct {
	svc       *service
	mra       *ramock.ResourceAccessor
	finishers []mock.Finisher
}

func TestPaymentRequestLookup(t *testing.T) {
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, &auth.Account{ID: "accountID", Type: auth.AccountType_PATIENT})
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{})
	paymentID := "paymentID"
	requestingEntityID := "requestingEntityID"
	customerEntityID := "customerEntityID"
	staticURLPrefix := "staticURLPrefix"
	cases := map[string]struct {
		TestParams  *testPaymentRequestParams
		Expected    interface{}
		ExpectedErr error
	}{
		"Success": {
			TestParams: func() *testPaymentRequestParams {
				mra := ramock.New(t)
				mra.Expect(mock.NewExpectation(mra.Payment, &payments.PaymentRequest{
					PaymentID: paymentID,
				}).WithReturns(&payments.PaymentResponse{
					Payment: &payments.Payment{
						ID:                 paymentID,
						RequestingEntityID: requestingEntityID,
						PaymentMethod: &payments.PaymentMethod{
							ID:          "ID",
							EntityID:    customerEntityID,
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
						Lifecycle:   payments.PAYMENT_LIFECYCLE_SUBMITTED,
						ChangeState: payments.PAYMENT_CHANGE_STATE_NONE,
					},
				}, nil))
				mra.Expect(mock.NewExpectation(mra.Entities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: requestingEntityID,
					},
				}).WithReturns([]*directory.Entity{
					{
						ID:   requestingEntityID,
						Info: &directory.EntityInfo{},
					},
				}, nil))
				return &testPaymentRequestParams{
					svc: &service{
						staticURLPrefix: staticURLPrefix,
					},
					mra:       mra,
					finishers: []mock.Finisher{mra},
				}
			}(),
			ExpectedErr: nil,
			Expected: &models.PaymentRequest{
				ID: paymentID,
				RequestingEntity: func() *models.Entity {
					ent, err := transformEntityToResponse(ctx, staticURLPrefix, &directory.Entity{
						ID:   requestingEntityID,
						Info: &directory.EntityInfo{},
					}, &device.SpruceHeaders{}, &auth.Account{ID: "accountID", Type: auth.AccountType_PATIENT})
					test.OK(t, err)
					return ent
				}(),
				Status: paymentStatus(payments.PAYMENT_LIFECYCLE_SUBMITTED, payments.PAYMENT_CHANGE_STATE_NONE),
				PaymentMethod: transformPaymentMethodToResponse(&payments.PaymentMethod{
					ID:          "ID",
					EntityID:    customerEntityID,
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
				}),
			},
		},
	}
	for cn, c := range cases {
		out, err := lookupPaymentRequest(ctx, c.TestParams.svc, c.TestParams.mra, paymentID)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, out)
		mock.FinishAll(c.TestParams.finishers...)
	}
}

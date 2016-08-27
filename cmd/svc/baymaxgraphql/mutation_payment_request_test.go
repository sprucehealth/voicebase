package main

import (
	"fmt"
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	ramock "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess/mock"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/textutil"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

type testAcceptPaymentRequestParams struct {
	p         graphql.ResolveParams
	finishers []mock.Finisher
}

func TestAcceptPaymentRequest(t *testing.T) {
	ctx := gqlctx.WithAccount(context.Background(), &auth.Account{ID: "accountID"})
	mutationID := "mutationID"
	requestinEntityID := "requestinEntityID"
	paymentRequestID := "paymentRequestID"
	paymentMethodID := "paymentMethodID"
	threadID := "threadID"
	entityID := "entityID"
	organizationID := "organizationID"
	cases := map[string]struct {
		In          acceptPaymentRequestInput
		TestParams  *testAcceptPaymentRequestParams
		Expected    interface{}
		ExpectedErr error
	}{
		"Success-NoThreadID": {
			In: acceptPaymentRequestInput{
				ClientMutationID: mutationID,
				PaymentRequestID: paymentRequestID,
				PaymentMethodID:  paymentMethodID,
			},
			TestParams: func() *testAcceptPaymentRequestParams {
				mra := ramock.New(t)
				mra.Expect(mock.NewExpectation(mra.Payment, &payments.PaymentRequest{
					PaymentID: paymentRequestID,
				}).WithReturns(&payments.PaymentResponse{
					Payment: &payments.Payment{
						ThreadID: threadID,
					},
				}, nil))
				mra.Expect(mock.NewExpectation(mra.Thread, threadID, "").WithReturns(&threading.Thread{
					ID:             threadID,
					OrganizationID: organizationID,
				}, nil))
				mra.Expect(mock.NewExpectation(mra.AcceptPayment, &payments.AcceptPaymentRequest{
					PaymentID:       paymentRequestID,
					PaymentMethodID: paymentMethodID,
				}).WithReturns(&payments.AcceptPaymentResponse{
					Payment: &payments.Payment{
						ID:                 paymentRequestID,
						RequestingEntityID: requestinEntityID,
						PaymentMethod: &payments.PaymentMethod{
							EntityID: entityID,
						},
					},
				}, nil))
				mra.Expect(mock.NewExpectation(mra.Entities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: requestinEntityID,
					},
				}).WithReturns([]*directory.Entity{{ID: requestinEntityID, Info: &directory.EntityInfo{}}}, nil))
				return &testAcceptPaymentRequestParams{
					p: graphql.ResolveParams{
						Context: ctx,
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: mra,
								"service": &service{
									staticURLPrefix: "staticURLPrefix",
								},
							},
						},
					},
					finishers: []mock.Finisher{mra},
				}
			}(),
			ExpectedErr: nil,
			Expected: func() *acceptPaymentRequestOutput {
				ent, err := transformEntityToResponse(ctx, "staticURLPrefix", &directory.Entity{ID: requestinEntityID, Info: &directory.EntityInfo{}}, devicectx.SpruceHeaders(ctx), gqlctx.Account(ctx))
				test.OK(t, err)
				return &acceptPaymentRequestOutput{
					ClientMutationID: mutationID,
					Success:          true,
					PaymentRequest: &models.PaymentRequest{
						ID:               paymentRequestID,
						RequestingEntity: ent,
						PaymentMethod: transformPaymentMethodToResponse(&payments.PaymentMethod{
							EntityID: entityID,
						}),
					},
				}
			}(),
		},
		"Success-ThreadID": {
			In: acceptPaymentRequestInput{
				ClientMutationID: mutationID,
				PaymentRequestID: paymentRequestID,
				PaymentMethodID:  paymentMethodID,
			},
			TestParams: func() *testAcceptPaymentRequestParams {
				mra := ramock.New(t)
				mra.Expect(mock.NewExpectation(mra.Payment, &payments.PaymentRequest{
					PaymentID: paymentRequestID,
				}).WithReturns(&payments.PaymentResponse{
					Payment: &payments.Payment{
						ID:       paymentRequestID,
						ThreadID: threadID,
					},
				}, nil))
				mra.Expect(mock.NewExpectation(mra.Thread, threadID, "").WithReturns(&threading.Thread{
					ID:             threadID,
					OrganizationID: organizationID,
				}, nil))
				mra.Expect(mock.NewExpectation(mra.AcceptPayment, &payments.AcceptPaymentRequest{
					PaymentID:       paymentRequestID,
					PaymentMethodID: paymentMethodID,
				}).WithReturns(&payments.AcceptPaymentResponse{
					Payment: &payments.Payment{
						ID:                 paymentRequestID,
						ThreadID:           threadID,
						RequestingEntityID: requestinEntityID,
						PaymentMethod: &payments.PaymentMethod{
							EntityID: entityID,
						},
						Amount: 234,
					},
				}, nil))
				var title bml.BML
				title = append(title, "Completed Payment: ")
				title = append(title, &bml.Anchor{
					HREF: deeplink.PaymentURL("webDomain", organizationID, threadID, paymentRequestID),
					Text: "$" + textutil.FormatCurrencyAmount(fmt.Sprintf("%.2f", float64(234)/float64(100))),
				})
				titleText, err := title.Format()
				test.OK(t, err)
				summary, err := title.PlainText()
				test.OK(t, err)
				mra.Expect(mock.NewExpectation(mra.PostMessage, &threading.PostMessageRequest{
					UUID:     `accept_` + paymentRequestID,
					ThreadID: threadID,
					// TODO: For now just assume whoever owns the payment method accepted it
					FromEntityID: entityID,
					Title:        titleText,
					Summary:      summary,
				}))
				mra.Expect(mock.NewExpectation(mra.Entities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: requestinEntityID,
					},
				}).WithReturns([]*directory.Entity{{ID: requestinEntityID, Info: &directory.EntityInfo{}}}, nil))
				return &testAcceptPaymentRequestParams{
					p: graphql.ResolveParams{
						Context: ctx,
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: mra,
								"service": &service{
									staticURLPrefix: "staticURLPrefix",
									webDomain:       "webDomain",
								},
							},
						},
					},
					finishers: []mock.Finisher{mra},
				}
			}(),
			ExpectedErr: nil,
			Expected: func() *acceptPaymentRequestOutput {
				ent, err := transformEntityToResponse(ctx, "staticURLPrefix", &directory.Entity{ID: requestinEntityID, Info: &directory.EntityInfo{}}, devicectx.SpruceHeaders(ctx), gqlctx.Account(ctx))
				test.OK(t, err)
				return &acceptPaymentRequestOutput{
					ClientMutationID: mutationID,
					Success:          true,
					PaymentRequest: &models.PaymentRequest{
						ID:               paymentRequestID,
						RequestingEntity: ent,
						PaymentMethod: transformPaymentMethodToResponse(&payments.PaymentMethod{
							EntityID: entityID,
						}),
						AmountInCents: 234,
						Status:        "",
					},
				}
			}(),
		},
	}
	for cn, c := range cases {
		out, err := acceptPaymentRequest(c.TestParams.p, c.In)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, out)
		mock.FinishAll(c.TestParams.finishers...)
	}
}

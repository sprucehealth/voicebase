package server

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/testutil"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/payments"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type tServer struct {
	srv       payments.PaymentsServer
	finishers []mock.Finisher
}

func TestVendorAccounts(t *testing.T) {
	ctx := context.Background()
	stripeSecretKey := "stripeSecretKey"
	entityID := "entityID"
	id1, err := dal.NewVendorAccountID()
	test.OK(t, err)
	id2, err := dal.NewVendorAccountID()
	test.OK(t, err)
	cases := map[string]struct {
		Server      *tServer
		Request     *payments.VendorAccountsRequest
		Expected    *payments.VendorAccountsResponse
		ExpectedErr error
	}{
		"Error-EntityIDRequired": {
			Server: func() *tServer {
				srv, err := New(testutil.NewMockDAL(t), stripeSecretKey)
				test.OK(t, err)
				return &tServer{
					srv: srv,
				}
			}(),
			Request:     &payments.VendorAccountsRequest{},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "EntityID required"),
		},
		"Success": {
			Server: func() *tServer {
				mdal := testutil.NewMockDAL(t)
				mdal.Expect(mock.NewExpectation(mdal.EntityVendorAccounts, entityID).WithReturns([]*dal.VendorAccount{
					{
						ID: id1,
					},
					{
						ID: id2,
					},
				}, nil))
				srv, err := New(mdal, stripeSecretKey)
				test.OK(t, err)
				return &tServer{
					srv:       srv,
					finishers: []mock.Finisher{mdal},
				}
			}(),
			Request: &payments.VendorAccountsRequest{
				EntityID: entityID,
			},
			Expected: &payments.VendorAccountsResponse{
				VendorAccounts: transformVendorAccountsToResponse([]*dal.VendorAccount{
					{
						ID: id1,
					},
					{
						ID: id2,
					},
				}),
			},
			ExpectedErr: nil,
		},
	}
	for cn, c := range cases {
		resp, err := c.Server.srv.VendorAccounts(ctx, c.Request)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, resp)
		mock.FinishAll(c.Server.finishers...)
	}
}

func TestConnectVendorAccount(t *testing.T) {
	ctx := context.Background()
	entityID := "entityID"
	code := "accessCode"
	cases := map[string]struct {
		Server      *tServer
		Request     *payments.ConnectVendorAccountRequest
		Expected    *payments.ConnectVendorAccountResponse
		ExpectedErr error
	}{
		"Error-EntityIDRequired": {
			Server: func() *tServer {
				srv, err := New(nil, "")
				test.OK(t, err)
				return &tServer{
					srv: srv,
				}
			}(),
			Request:     &payments.ConnectVendorAccountRequest{},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "EntityID required"),
		},
		"Error-UnknownAccountType": {
			Server: func() *tServer {
				srv, err := New(nil, "")
				test.OK(t, err)
				return &tServer{
					srv: srv,
				}
			}(),
			Request: &payments.ConnectVendorAccountRequest{
				EntityID: entityID,
			},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "Unsupported vendor account type %s", payments.VENDOR_ACCOUNT_TYPE_UNKNOWN),
		},
		"Success-Stripe": {
			Server: func() *tServer {
				mSOAuth := testutil.NewMockStripeOAuth(t)
				mSOAuth.Expect(mock.NewExpectation(mSOAuth.RequestStripeAccessToken, code).WithReturns(&oauth.StripeAccessTokenResponse{
					AccessToken:          "AccessToken",
					RefreshToken:         "RefreshToken",
					StripePublishableKey: "PublishableKey",
					StripeUserID:         "ConnectedAccountID",
					Scope:                "Scope",
					LiveMode:             true,
				}, nil))

				mdal := testutil.NewMockDAL(t)
				mdal.Expect(mock.NewExpectation(mdal.InsertVendorAccount, &dal.VendorAccount{
					AccessToken:        "AccessToken",
					RefreshToken:       "RefreshToken",
					PublishableKey:     "PublishableKey",
					ConnectedAccountID: "ConnectedAccountID",
					Scope:              "Scope",
					Live:               true,
					AccountType:        dal.VendorAccountAccountTypeStripe,
					Lifecycle:          dal.VendorAccountLifecycleConnected,
					ChangeState:        dal.VendorAccountChangeStateNone,
					EntityID:           entityID,
				}))
				mdal.Expect(mock.NewExpectation(mdal.EntityVendorAccounts, entityID).WithReturns([]*dal.VendorAccount{
					{
						AccessToken:        "AccessToken",
						RefreshToken:       "RefreshToken",
						PublishableKey:     "PublishableKey",
						ConnectedAccountID: "ConnectedAccountID",
						Scope:              "Scope",
						Live:               true,
						AccountType:        dal.VendorAccountAccountTypeStripe,
						Lifecycle:          dal.VendorAccountLifecycleConnected,
						ChangeState:        dal.VendorAccountChangeStateNone,
						EntityID:           entityID,
					},
				}, nil))

				srv, err := New(mdal, "")
				test.OK(t, err)
				ssrv := srv.(*server)
				ssrv.stripeOAuth = mSOAuth
				return &tServer{
					srv:       ssrv,
					finishers: []mock.Finisher{mSOAuth, mdal},
				}
			}(),
			Request: &payments.ConnectVendorAccountRequest{
				EntityID:          entityID,
				VendorAccountType: payments.VENDOR_ACCOUNT_TYPE_STRIPE,
				ConnectVendorAccountOneof: &payments.ConnectVendorAccountRequest_StripeRequest{
					StripeRequest: &payments.StripeAccountConnectRequest{
						Code: code,
					},
				},
			},
			Expected: &payments.ConnectVendorAccountResponse{
				VendorAccounts: transformVendorAccountsToResponse([]*dal.VendorAccount{
					{
						AccessToken:        "AccessToken",
						RefreshToken:       "RefreshToken",
						PublishableKey:     "PublishableKey",
						ConnectedAccountID: "ConnectedAccountID",
						Scope:              "Scope",
						Live:               true,
						AccountType:        dal.VendorAccountAccountTypeStripe,
						Lifecycle:          dal.VendorAccountLifecycleConnected,
						ChangeState:        dal.VendorAccountChangeStateNone,
						EntityID:           entityID,
					},
				}),
			},
			ExpectedErr: nil,
		},
	}
	for cn, c := range cases {
		resp, err := c.Server.srv.ConnectVendorAccount(ctx, c.Request)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, resp)
		mock.FinishAll(c.Server.finishers...)
	}
}

func TestDisconnectVendorAccount(t *testing.T) {
	ctx := context.Background()
	id, err := dal.NewVendorAccountID()
	test.OK(t, err)
	cases := map[string]struct {
		Server      *tServer
		Request     *payments.DisconnectVendorAccountRequest
		Expected    *payments.DisconnectVendorAccountResponse
		ExpectedErr error
	}{
		"Error-VendorAccountID": {
			Server: func() *tServer {
				srv, err := New(testutil.NewMockDAL(t), "")
				test.OK(t, err)
				return &tServer{
					srv: srv,
				}
			}(),
			Request:     &payments.DisconnectVendorAccountRequest{},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "VendorAccountID required"),
		},
		"Success-Stripe-AlreadyDisconnected": {
			Server: func() *tServer {
				mdal := testutil.NewMockDAL(t)
				mdal.Expect(mock.NewExpectation(mdal.VendorAccount, id, []dal.QueryOption{dal.ForUpdate}).WithReturns(&dal.VendorAccount{
					ID:        id,
					Lifecycle: dal.VendorAccountLifecycleDisconnected,
				}, nil))
				srv, err := New(mdal, "")
				test.OK(t, err)
				return &tServer{
					srv:       srv,
					finishers: []mock.Finisher{mdal},
				}
			}(),
			Request: &payments.DisconnectVendorAccountRequest{
				VendorAccountID: id.String(),
			},
			Expected:    &payments.DisconnectVendorAccountResponse{},
			ExpectedErr: nil,
		},
		"Success-Stripe": {
			Server: func() *tServer {
				mdal := testutil.NewMockDAL(t)
				mdal.Expect(mock.NewExpectation(mdal.VendorAccount, id, []dal.QueryOption{dal.ForUpdate}).WithReturns(&dal.VendorAccount{
					ID:        id,
					Lifecycle: dal.VendorAccountLifecycleConnected,
				}, nil))
				mdal.Expect(mock.NewExpectation(mdal.UpdateVendorAccount, id, &dal.VendorAccountUpdate{
					Lifecycle:   dal.VendorAccountLifecycleDisconnected,
					ChangeState: dal.VendorAccountChangeStatePending,
				}))
				srv, err := New(mdal, "")
				test.OK(t, err)
				return &tServer{
					srv:       srv,
					finishers: []mock.Finisher{mdal},
				}
			}(),
			Request: &payments.DisconnectVendorAccountRequest{
				VendorAccountID: id.String(),
			},
			Expected:    &payments.DisconnectVendorAccountResponse{},
			ExpectedErr: nil,
		},
	}
	for cn, c := range cases {
		resp, err := c.Server.srv.DisconnectVendorAccount(ctx, c.Request)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, resp)
		mock.FinishAll(c.Server.finishers...)
	}
}

package server

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	istripe "github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/testutil"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	dmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/stripe/stripe-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type tServer struct {
	srv         payments.PaymentsServer
	mdal        *testutil.MockDAL
	mdir        *dmock.Client
	mstripe     *testutil.MockIdempotentStripeClient
	otherFinish []mock.Finisher
}

func (t *tServer) Finishers() []mock.Finisher {
	return append([]mock.Finisher{t.mdal, t.mdir, t.mstripe}, t.otherFinish...)
}

func (t *tServer) AddFinisher(f mock.Finisher) {
	t.otherFinish = append(t.otherFinish, f)
}

func newTestServer(t *testing.T, masterVendorAccount *dal.VendorAccount, stripeKey string) *tServer {
	mdal := testutil.NewMockDAL(t)
	mstripe := testutil.NewMockIdempotentStripeClient(t)
	mdir := dmock.New(t)
	masterVendorAccount.AccessToken = stripeKey
	mdal.Expect(mock.NewExpectation(mdal.VendorAccount, masterVendorAccount.ID).WithReturns(masterVendorAccount, nil))
	mstripe.Expect(mock.NewExpectation(mstripe.Account))
	srv, err := New(mdal, mdir, masterVendorAccount.ID.String(), mstripe, stripeKey)
	test.OK(t, err)
	return &tServer{
		srv:     srv,
		mdal:    mdal,
		mdir:    mdir,
		mstripe: mstripe,
	}
}

func TestVendorAccounts(t *testing.T) {
	ctx := context.Background()
	stripeSecretKey := "stripeSecretKey"
	entityID := "entityID"
	id1, err := dal.NewVendorAccountID()
	test.OK(t, err)
	id2, err := dal.NewVendorAccountID()
	test.OK(t, err)
	masterVendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	cases := map[string]struct {
		Server      *tServer
		Request     *payments.VendorAccountsRequest
		Expected    *payments.VendorAccountsResponse
		ExpectedErr error
	}{
		"Error-EntityIDRequired": {
			Server: func() *tServer {
				return newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
			}(),
			Request:     &payments.VendorAccountsRequest{},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "EntityID required"),
		},
		"Success": {
			Server: func() *tServer {
				tsrv := newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.EntityVendorAccounts, entityID).WithReturns([]*dal.VendorAccount{
					{
						ID: id1,
					},
					{
						ID: id2,
					},
				}, nil))
				return tsrv
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
		mock.FinishAll(c.Server.Finishers()...)
	}
}

func TestConnectVendorAccount(t *testing.T) {
	ctx := context.Background()
	entityID := "entityID"
	code := "accessCode"
	stripeSecretKey := "stripeSecretKey"
	masterVendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	cases := map[string]struct {
		Server      *tServer
		Request     *payments.ConnectVendorAccountRequest
		Expected    *payments.ConnectVendorAccountResponse
		ExpectedErr error
	}{
		"Error-EntityIDRequired": {
			Server: func() *tServer {
				return newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
			}(),
			Request:     &payments.ConnectVendorAccountRequest{},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "EntityID required"),
		},
		"Error-UnknownAccountType": {
			Server: func() *tServer {
				return newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
			}(),
			Request: &payments.ConnectVendorAccountRequest{
				EntityID: entityID,
			},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "Unsupported vendor account type %s", payments.VENDOR_ACCOUNT_TYPE_UNKNOWN),
		},
		"Success-Stripe": {
			Server: func() *tServer {
				tsrv := newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
				mSOAuth := testutil.NewMockStripeOAuth(t)
				mSOAuth.Expect(mock.NewExpectation(mSOAuth.RequestStripeAccessToken, code).WithReturns(&oauth.StripeAccessTokenResponse{
					AccessToken:          "AccessToken",
					RefreshToken:         "RefreshToken",
					StripePublishableKey: "PublishableKey",
					StripeUserID:         "ConnectedAccountID",
					Scope:                "Scope",
					LiveMode:             true,
				}, nil))
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.InsertVendorAccount, &dal.VendorAccount{
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
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.EntityVendorAccounts, entityID).WithReturns([]*dal.VendorAccount{
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
				ssrv := tsrv.srv.(*server)
				ssrv.stripeOAuth = mSOAuth
				tsrv.srv = ssrv
				tsrv.AddFinisher(mSOAuth)
				return tsrv
			}(),
			Request: &payments.ConnectVendorAccountRequest{
				EntityID: entityID,
				Type:     payments.VENDOR_ACCOUNT_TYPE_STRIPE,
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
		mock.FinishAll(c.Server.Finishers()...)
	}
}

func TestUpdateVendorAccount(t *testing.T) {
	ctx := context.Background()
	id, err := dal.NewVendorAccountID()
	test.OK(t, err)
	masterVendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	stripeSecretKey := "stripeSecretKey"
	cases := map[string]struct {
		Server      *tServer
		Request     *payments.UpdateVendorAccountRequest
		Expected    *payments.UpdateVendorAccountResponse
		ExpectedErr error
	}{
		"Error-VendorAccountID": {
			Server: func() *tServer {
				return newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
			}(),
			Request:     &payments.UpdateVendorAccountRequest{},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "VendorAccountID required"),
		},
		"Success-Stripe-AlreadyInState": {
			Server: func() *tServer {
				tsrv := newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.VendorAccount, id, []dal.QueryOption{dal.ForUpdate}).WithReturns(&dal.VendorAccount{
					ID:          id,
					Lifecycle:   dal.VendorAccountLifecycleDisconnected,
					ChangeState: dal.VendorAccountChangeStatePending,
				}, nil))
				return tsrv
			}(),
			Request: &payments.UpdateVendorAccountRequest{
				VendorAccountID: id.String(),
				Lifecycle:       payments.VENDOR_ACCOUNT_LIFECYCLE_DISCONNECTED,
				ChangeState:     payments.VENDOR_ACCOUNT_CHANGE_STATE_PENDING,
			},
			Expected:    &payments.UpdateVendorAccountResponse{},
			ExpectedErr: nil,
		},
		"Success-Stripe": {
			Server: func() *tServer {
				tsrv := newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.VendorAccount, id, []dal.QueryOption{dal.ForUpdate}).WithReturns(&dal.VendorAccount{
					ID:          id,
					Lifecycle:   dal.VendorAccountLifecycleConnected,
					ChangeState: dal.VendorAccountChangeStateNone,
				}, nil))
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.UpdateVendorAccount, id, &dal.VendorAccountUpdate{
					Lifecycle:   dal.VendorAccountLifecycleDisconnected,
					ChangeState: dal.VendorAccountChangeStatePending,
				}))
				return tsrv
			}(),
			Request: &payments.UpdateVendorAccountRequest{
				VendorAccountID: id.String(),
				Lifecycle:       payments.VENDOR_ACCOUNT_LIFECYCLE_DISCONNECTED,
				ChangeState:     payments.VENDOR_ACCOUNT_CHANGE_STATE_PENDING,
			},
			Expected:    &payments.UpdateVendorAccountResponse{},
			ExpectedErr: nil,
		},
	}
	for cn, c := range cases {
		resp, err := c.Server.srv.UpdateVendorAccount(ctx, c.Request)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, resp)
		mock.FinishAll(c.Server.Finishers()...)
	}
}

func TestPaymentMethods(t *testing.T) {
	ctx := context.Background()
	pmID1, err := dal.NewPaymentMethodID()
	test.OK(t, err)
	pmID2, err := dal.NewPaymentMethodID()
	test.OK(t, err)
	cID, err := dal.NewCustomerID()
	test.OK(t, err)
	masterVendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	stripeSecretKey := "stripeSecretKey"
	entityID := "entityID"
	storageID := "storageID"
	cardID := "cardID"
	cases := map[string]struct {
		Server      *tServer
		Request     *payments.PaymentMethodsRequest
		Expected    *payments.PaymentMethodsResponse
		ExpectedErr error
	}{
		"Error-EntityID": {
			Server: func() *tServer {
				return newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
			}(),
			Request:     &payments.PaymentMethodsRequest{},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "EntityID required"),
		},
		"Success": {
			Server: func() *tServer {
				tsrv := newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.EntityPaymentMethods, masterVendorAccountID, entityID, []dal.QueryOption(nil)).WithReturns(
					[]*dal.PaymentMethod{
						{
							ID:          pmID1,
							CustomerID:  cID,
							EntityID:    entityID,
							Lifecycle:   dal.PaymentMethodLifecycleActive,
							ChangeState: dal.PaymentMethodChangeStateNone,
							StorageType: dal.PaymentMethodStorageTypeStripe,
							StorageID:   storageID,
						},
						{
							ID:          pmID2,
							CustomerID:  cID,
							EntityID:    entityID,
							Lifecycle:   dal.PaymentMethodLifecycleActive,
							ChangeState: dal.PaymentMethodChangeStateNone,
							StorageType: dal.PaymentMethodStorageTypeStripe,
							StorageID:   storageID,
						},
					}, nil))
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.CustomerForVendor, masterVendorAccountID, entityID, []dal.QueryOption(nil)).WithReturns(
					&dal.Customer{
						ID:        cID,
						StorageID: storageID,
					}, nil))
				tsrv.mstripe.Expect(mock.NewExpectation(tsrv.mstripe.Card, storageID, &stripe.CardParams{
					Customer: storageID,
				}).WithReturns(&stripe.Card{
					ID:                 cardID,
					TokenizationMethod: stripe.TokenizationMethod("TokenizationMethod"),
					Brand:              stripe.CardBrand("Brand"),
					LastFour:           "LastFour",
				}, nil))
				tsrv.mstripe.Expect(mock.NewExpectation(tsrv.mstripe.Card, storageID, &stripe.CardParams{
					Customer: storageID,
				}).WithReturns(&stripe.Card{
					ID:                 cardID,
					TokenizationMethod: stripe.TokenizationMethod("TokenizationMethod"),
					Brand:              stripe.CardBrand("Brand"),
					LastFour:           "LastFour",
				}, nil))
				return tsrv
			}(),
			Request: &payments.PaymentMethodsRequest{
				EntityID: entityID,
			},
			Expected: &payments.PaymentMethodsResponse{
				PaymentMethods: []*payments.PaymentMethod{
					{
						ID:          pmID1.String(),
						EntityID:    entityID,
						Default:     true,
						Lifecycle:   payments.PAYMENT_METHOD_LIFECYCLE_ACTIVE,
						ChangeState: payments.PAYMENT_METHOD_CHANGE_STATE_NONE,
						StorageType: payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE,
						Type:        payments.PAYMENT_METHOD_TYPE_CARD,
						PaymentMethodOneof: &payments.PaymentMethod_StripeCard{
							StripeCard: &payments.StripeCard{
								ID:                 cardID,
								TokenizationMethod: "TokenizationMethod",
								Brand:              "Brand",
								Last4:              "LastFour",
							},
						},
					},
					{
						ID:          pmID2.String(),
						EntityID:    entityID,
						Default:     false,
						Lifecycle:   payments.PAYMENT_METHOD_LIFECYCLE_ACTIVE,
						ChangeState: payments.PAYMENT_METHOD_CHANGE_STATE_NONE,
						StorageType: payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE,
						Type:        payments.PAYMENT_METHOD_TYPE_CARD,
						PaymentMethodOneof: &payments.PaymentMethod_StripeCard{
							StripeCard: &payments.StripeCard{
								ID:                 cardID,
								TokenizationMethod: "TokenizationMethod",
								Brand:              "Brand",
								Last4:              "LastFour",
							},
						},
					},
				},
			},
			ExpectedErr: nil,
		},
	}
	for cn, c := range cases {
		resp, err := c.Server.srv.PaymentMethods(ctx, c.Request)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, resp)
		mock.FinishAll(c.Server.Finishers()...)
	}
}

func TestDeletePaymentMethod(t *testing.T) {
	ctx := context.Background()
	pmID1, err := dal.NewPaymentMethodID()
	test.OK(t, err)
	pmID2, err := dal.NewPaymentMethodID()
	test.OK(t, err)
	cID, err := dal.NewCustomerID()
	test.OK(t, err)
	masterVendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	stripeSecretKey := "stripeSecretKey"
	entityID := "entityID"
	storageID := "storageID"
	cardID := "cardID"
	connectedAccountID := "connectedAccountID"
	storageFingerprint := "storageFingerprint"
	cases := map[string]struct {
		Server      *tServer
		Request     *payments.DeletePaymentMethodRequest
		Expected    *payments.DeletePaymentMethodResponse
		ExpectedErr error
	}{
		"Error-PaymentMethodID": {
			Server: func() *tServer {
				return newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
			}(),
			Request:     &payments.DeletePaymentMethodRequest{},
			Expected:    nil,
			ExpectedErr: grpc.Errorf(codes.InvalidArgument, "PaymentMethodID required"),
		},
		"Success": {
			Server: func() *tServer {
				tsrv := newTestServer(t, &dal.VendorAccount{ID: masterVendorAccountID}, stripeSecretKey)
				// Delete Payment Methods
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.PaymentMethod, pmID1, []dal.QueryOption(nil)).WithReturns(
					&dal.PaymentMethod{
						ID:                 pmID1,
						CustomerID:         cID,
						EntityID:           entityID,
						VendorAccountID:    masterVendorAccountID,
						Lifecycle:          dal.PaymentMethodLifecycleActive,
						ChangeState:        dal.PaymentMethodChangeStateNone,
						StorageType:        dal.PaymentMethodStorageTypeStripe,
						StorageID:          storageID,
						StorageFingerprint: storageFingerprint,
					}, nil))
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.PaymentMethod, pmID1, []dal.QueryOption{dal.ForUpdate}).WithReturns(
					&dal.PaymentMethod{
						ID:                 pmID1,
						CustomerID:         cID,
						EntityID:           entityID,
						VendorAccountID:    masterVendorAccountID,
						Lifecycle:          dal.PaymentMethodLifecycleActive,
						ChangeState:        dal.PaymentMethodChangeStateNone,
						StorageType:        dal.PaymentMethodStorageTypeStripe,
						StorageID:          storageID,
						StorageFingerprint: storageFingerprint,
					}, nil))
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.VendorAccount, masterVendorAccountID).WithReturns(
					&dal.VendorAccount{
						ID:                 masterVendorAccountID,
						ConnectedAccountID: connectedAccountID,
					}, nil))
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.Customer, cID, []dal.QueryOption(nil)).WithReturns(
					&dal.Customer{
						ID:        cID,
						StorageID: storageID,
					}, nil))
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.UpdatePaymentMethod, pmID1, &dal.PaymentMethodUpdate{
					Lifecycle:   dal.PaymentMethodLifecycleDeleted,
					ChangeState: dal.PaymentMethodChangeStateNone,
				}))
				tsrv.mstripe.Expect(mock.NewExpectation(tsrv.mstripe.DeleteCard, storageID, &stripe.CardParams{
					Customer: storageID,
					Params: stripe.Params{
						StripeAccount: connectedAccountID,
					},
				}, []istripe.CallOption(nil)))
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.PaymentMethodsWithFingerprint, storageFingerprint, []dal.QueryOption(nil)))

				// Return Existing Payment Methods
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.EntityPaymentMethods, masterVendorAccountID, entityID, []dal.QueryOption(nil)).WithReturns(
					[]*dal.PaymentMethod{
						{
							ID:          pmID2,
							CustomerID:  cID,
							EntityID:    entityID,
							Lifecycle:   dal.PaymentMethodLifecycleActive,
							ChangeState: dal.PaymentMethodChangeStateNone,
							StorageType: dal.PaymentMethodStorageTypeStripe,
							StorageID:   storageID,
						},
					}, nil))
				tsrv.mdal.Expect(mock.NewExpectation(tsrv.mdal.CustomerForVendor, masterVendorAccountID, entityID, []dal.QueryOption(nil)).WithReturns(
					&dal.Customer{
						ID:        cID,
						StorageID: storageID,
					}, nil))
				tsrv.mstripe.Expect(mock.NewExpectation(tsrv.mstripe.Card, storageID, &stripe.CardParams{
					Customer: storageID,
				}).WithReturns(&stripe.Card{
					ID:                 cardID,
					TokenizationMethod: stripe.TokenizationMethod("TokenizationMethod"),
					Brand:              stripe.CardBrand("Brand"),
					LastFour:           "LastFour",
				}, nil))
				return tsrv
			}(),
			Request: &payments.DeletePaymentMethodRequest{
				PaymentMethodID: pmID1.String(),
			},
			Expected: &payments.DeletePaymentMethodResponse{
				PaymentMethods: []*payments.PaymentMethod{
					{
						ID:          pmID2.String(),
						EntityID:    entityID,
						Default:     true,
						Lifecycle:   payments.PAYMENT_METHOD_LIFECYCLE_ACTIVE,
						ChangeState: payments.PAYMENT_METHOD_CHANGE_STATE_NONE,
						StorageType: payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE,
						Type:        payments.PAYMENT_METHOD_TYPE_CARD,
						PaymentMethodOneof: &payments.PaymentMethod_StripeCard{
							StripeCard: &payments.StripeCard{
								ID:                 cardID,
								TokenizationMethod: "TokenizationMethod",
								Brand:              "Brand",
								Last4:              "LastFour",
							},
						},
					},
				},
			},
			ExpectedErr: nil,
		},
	}
	for cn, c := range cases {
		resp, err := c.Server.srv.DeletePaymentMethod(ctx, c.Request)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, resp)
		mock.FinishAll(c.Server.Finishers()...)
	}
}

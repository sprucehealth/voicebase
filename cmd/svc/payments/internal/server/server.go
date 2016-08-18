package server

import (
	"fmt"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	istripe "github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/stripe/stripe-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrf = grpc.Errorf

func grpcErrorf(c codes.Code, format string, a ...interface{}) error {
	if c == codes.Internal {
		golog.LogDepthf(1, golog.ERR, "Payments - Internal GRPC Error: %s", fmt.Sprintf(format, a...))
	}
	return grpcErrf(c, format, a...)
}

func grpcError(err error) error {
	if grpc.Code(err) == codes.Unknown {
		return grpcErrorf(codes.Internal, err.Error())
	}
	return err
}

func grpcIErrorf(fmt string, args ...interface{}) error {
	golog.LogDepthf(1, golog.ERR, fmt, args...)
	return grpcErrorf(codes.Internal, fmt, args...)
}

var (
	// ErrNotImplemented is returned from RPC calls that have yet to be implemented
	ErrNotImplemented = errors.New("Not Implemented")
)

type server struct {
	dal             dal.DAL
	directoryClient directory.DirectoryClient
	// The master vendor account owns all customers and payment methods and adds applicable ones to individual vendor accounts
	masterVendorAccount *dal.VendorAccount
	stripeOAuth         oauth.StripeOAuth
	stripeClient        istripe.IdempotentStripeClient
}

// New returns an initialized instance of server after performing initial validation
func New(dl dal.DAL,
	directoryClient directory.DirectoryClient,
	sMasterVendorAccountID string,
	stripeClient istripe.IdempotentStripeClient,
	stripeSecretKey string) (payments.PaymentsServer, error) {
	masterVendorAccountID, err := dal.ParseVendorAccountID(sMasterVendorAccountID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	masterVendorAccount, err := validateMasterVendorAccount(dl, masterVendorAccountID, stripeSecretKey, stripeClient)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &server{
		dal:                 dl,
		directoryClient:     directoryClient,
		masterVendorAccount: masterVendorAccount,
		stripeOAuth:         oauth.NewStripe(stripeSecretKey, ""),
		stripeClient:        stripeClient,
	}, nil
}

func validateMasterVendorAccount(dl dal.DAL, masterVendorAccountID dal.VendorAccountID, stripeSecretKey string, stripeClient istripe.IdempotentStripeClient) (*dal.VendorAccount, error) {
	ctx := context.Background()
	masterVendorAccount, err := dl.VendorAccount(ctx, masterVendorAccountID)
	if err != nil {
		return nil, errors.Errorf("Failed to validate master vendor account id: %s - %s", masterVendorAccountID, err)
	}
	// Add a little extra certainty in that we provide the key to double check our account
	if masterVendorAccount.AccessToken != stripeSecretKey {
		return nil, errors.Errorf("The provided stripe secret key does not match the stored value mapped to %s", masterVendorAccountID)
	}
	masterStripeAccount, err := stripeClient.Account(context.Background())
	if err != nil {
		return nil, errors.Errorf("Encountered an error when validating Stripe credentials: %s", err)
	}
	golog.Infof("Master Stripe Account: %+v", masterStripeAccount)
	return masterVendorAccount, nil
}

func (s *server) CreatePaymentMethod(ctx context.Context, req *payments.CreatePaymentMethodRequest) (*payments.CreatePaymentMethodResponse, error) {
	if req.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "EntityID required")
	}
	customer, err := s.addCustomer(ctx, s.masterVendorAccount, req.EntityID)
	if err != nil {
		return nil, grpcError(err)
	}
	var token string
	switch req.StorageType {
	case payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE:
		switch req.Type {
		case payments.PAYMENT_METHOD_TYPE_CARD:
			stripeCard := req.GetStripeCard()
			if stripeCard.Token == "" {
				return nil, grpcErrorf(codes.InvalidArgument, "Token required")
			}
			token = stripeCard.Token
		default:
			return nil, errors.Errorf("Unhandled payment method type %s for customer %s payment method addition", req.Type, customer.ID)
		}
	default:
		return nil, grpcErrorf(codes.InvalidArgument, "Unhandled payment method storage type %s", req.StorageType)
	}
	_, err = s.addPaymentMethod(ctx, s.masterVendorAccount, customer, req.Type, token)
	if err != nil {
		return nil, grpcError(err)
	}
	resp, err := s.PaymentMethods(ctx, &payments.PaymentMethodsRequest{EntityID: req.EntityID})
	if err != nil {
		return nil, grpcError(err)
	}
	return &payments.CreatePaymentMethodResponse{
		PaymentMethods: resp.PaymentMethods,
	}, nil
}

func (s *server) addCustomer(ctx context.Context, vendorAccount *dal.VendorAccount, entityID string) (*dal.Customer, error) {
	// Check to see if we've already added this customer
	customer, err := s.dal.CustomerForVendor(ctx, vendorAccount.ID, entityID)
	if err != nil && errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	} else if customer != nil {
		golog.Debugf("Customer FOUND - Entity: %s for VendorAccount: %s NOT ADDING", entityID, vendorAccount.ID)
		return customer, nil
	}

	// If we haven't added this customer look up the information we will want to associate with them
	ent, err := directory.SingleEntity(ctx, s.directoryClient, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	})
	if errors.Cause(err) == directory.ErrEntityNotFound {
		return nil, grpcErrorf(codes.NotFound, "Entity %s Not Found", entityID)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	// TODO: In the future how do we know what email to use?
	var customerEmail string
	for _, c := range ent.Contacts {
		if c.ContactType == directory.ContactType_EMAIL && !c.Provisioned {
			customerEmail = c.Value
			break
		}
	}
	if customerEmail == "" {
		// TODO: Is worth an error?
		golog.Errorf("Encountered payments customer addition for entity %s but could not find an associated unprovisioned email", ent.ID)
	}

	var newCustomer *dal.Customer
	switch vendorAccount.AccountType {
	case dal.VendorAccountAccountTypeStripe:
		stripeCustomer, err := s.stripeClient.CreateCustomer(ctx, &stripe.CustomerParams{
			Desc:  customerDescription(ent),
			Email: customerEmail,
			Params: stripe.Params{
				StripeAccount: vendorAccount.ConnectedAccountID,
				Meta:          map[string]string{"entity_id": entityID},
			},
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		newCustomer = &dal.Customer{
			StorageType: dal.CustomerStorageTypeStripe,
			StorageID:   stripeCustomer.ID,
		}
	default:
		return nil, errors.Errorf("Unknown vendor account type %s for vendor account %s customer creation", vendorAccount.AccountType, vendorAccount.ID)
	}
	// sanity
	if newCustomer == nil {
		return nil, grpcErrorf(codes.Internal, "nil newCustomer, this should never happen")
	}
	newCustomer.VendorAccountID = vendorAccount.ID
	newCustomer.EntityID = entityID
	newCustomer.Lifecycle = dal.CustomerLifecycleActive
	newCustomer.ChangeState = dal.CustomerChangeStateNone

	id, err := s.dal.InsertCustomer(ctx, newCustomer)
	if err != nil {
		return nil, errors.Trace(err)
	}
	newCustomer.ID = id
	golog.Debugf("Customer NOT FOUND - Entity: %s for VendorAccount: %s, ADDED - %+v", entityID, vendorAccount.ID, newCustomer)
	return newCustomer, nil
}

// TODO: What should be in this info?
func customerDescription(ent *directory.Entity) string {
	return ent.Info.DisplayName + " - Added by Spruce Health"
}

func (s *server) addPaymentMethod(ctx context.Context, vendorAccount *dal.VendorAccount, customer *dal.Customer, paymentMethodType payments.PaymentMethodType, token string) (*dal.PaymentMethod, error) {
	var newPaymentMethod *dal.PaymentMethod
	switch vendorAccount.AccountType {
	case dal.VendorAccountAccountTypeStripe:
		switch paymentMethodType {
		case payments.PAYMENT_METHOD_TYPE_CARD:
			stripeCard, err := s.stripeClient.CreateCard(ctx, &stripe.CardParams{
				Customer: customer.StorageID,
				Token:    token,
				Params: stripe.Params{
					StripeAccount: vendorAccount.ConnectedAccountID,
					Meta: map[string]string{
						"entity_id":   customer.EntityID,
						"customer_id": customer.ID.String(),
					},
				},
			})
			if err != nil {
				return nil, errors.Trace(err)
			}
			newPaymentMethod = &dal.PaymentMethod{
				StorageType:        dal.PaymentMethodStorageTypeStripe,
				StorageID:          stripeCard.ID,
				StorageFingerprint: stripeCard.Fingerprint,
			}
		default:
			return nil, errors.Errorf("Unhandled payment method type %s for vendor account %s payment method addition", paymentMethodType, vendorAccount.ID)
		}
	default:
		return nil, errors.Errorf("Unhandled vendor account type %s for vendor account %s payment method addition", vendorAccount.AccountType, vendorAccount.ID)
	}
	// sanity
	if newPaymentMethod == nil {
		return nil, grpcErrorf(codes.Internal, "nil newPaymentMethod, this should never happen")
	}
	newPaymentMethod.VendorAccountID = vendorAccount.ID
	newPaymentMethod.CustomerID = customer.ID
	newPaymentMethod.EntityID = customer.EntityID
	newPaymentMethod.Lifecycle = dal.PaymentMethodLifecycleActive
	newPaymentMethod.ChangeState = dal.PaymentMethodChangeStateNone

	// Check to see if we've already added this payment method - the stripe endpoint is idempotent
	paymentMethod, err := s.dal.PaymentMethodWithFingerprint(ctx, customer.ID, newPaymentMethod.StorageFingerprint)
	if err != nil && errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	} else if paymentMethod != nil {
		golog.Debugf("Payment Method FOUND - Fingerprint: %s Entity: %s for VendorAccount: %s - NOT ADDING", paymentMethod.StorageFingerprint, paymentMethod.EntityID, vendorAccount.ID)
		return paymentMethod, nil
	}

	golog.Debugf("Payment Method NOT FOUND - Fingerprint: %s Entity: %s for VendorAccount: %s - ADDING", newPaymentMethod.StorageFingerprint, newPaymentMethod.EntityID, vendorAccount.ID)
	id, err := s.dal.InsertPaymentMethod(ctx, newPaymentMethod)
	if err != nil {
		return nil, errors.Trace(err)
	}
	newPaymentMethod.ID = id
	return newPaymentMethod, nil
}

// TODO: Dedupe inserts on account ID in the event of multiple connections from same account
func (s *server) ConnectVendorAccount(ctx context.Context, req *payments.ConnectVendorAccountRequest) (*payments.ConnectVendorAccountResponse, error) {
	if req.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "EntityID required")
	}
	var vendorAccount *dal.VendorAccount
	switch req.Type {
	case payments.VENDOR_ACCOUNT_TYPE_STRIPE:
		stripeReq := req.GetStripeRequest()
		if stripeReq.Code == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "Code required")
		}
		accessTokenResponse, err := s.stripeOAuth.RequestStripeAccessToken(stripeReq.Code)
		if err != nil {
			return nil, grpcError(err)
		}
		vendorAccount = &dal.VendorAccount{
			AccessToken:        accessTokenResponse.AccessToken,
			RefreshToken:       accessTokenResponse.RefreshToken,
			PublishableKey:     accessTokenResponse.StripePublishableKey,
			ConnectedAccountID: accessTokenResponse.StripeUserID,
			Scope:              accessTokenResponse.Scope,
			Live:               accessTokenResponse.LiveMode,
			AccountType:        dal.VendorAccountAccountTypeStripe,
		}
	default:
		return nil, grpcErrorf(codes.InvalidArgument, "Unsupported vendor account type %s", req.Type)
	}
	// sanity
	if vendorAccount == nil {
		return nil, grpcErrorf(codes.Internal, "nil vendorAccount, this should never happen")
	}
	vendorAccount.Lifecycle = dal.VendorAccountLifecycleConnected
	vendorAccount.ChangeState = dal.VendorAccountChangeStateNone
	vendorAccount.EntityID = req.EntityID
	if _, err := s.dal.InsertVendorAccount(ctx, vendorAccount); err != nil {
		return nil, grpcError(err)
	}

	// Look up the new set of vendor accounts associated with the entity ID now
	vendorAccounts, err := s.VendorAccounts(ctx, &payments.VendorAccountsRequest{EntityID: req.EntityID})
	if err != nil {
		return nil, err
	}
	return &payments.ConnectVendorAccountResponse{
		VendorAccounts: vendorAccounts.VendorAccounts,
	}, nil
}

// TODO: This is perhaps something we could perhaps so more lazily with setting the record into PENDING/DELETING and having a worker clean it up.
// 	While I like that solution better, there are race conditions around deleting and readding the same card before the worker runs to consider.
//	Leave this synchronous for now.
func (s *server) DeletePaymentMethod(ctx context.Context, req *payments.DeletePaymentMethodRequest) (*payments.DeletePaymentMethodResponse, error) {
	if req.PaymentMethodID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "PaymentMethodID required")
	}
	paymentMethodID, err := dal.ParsePaymentMethodID(req.PaymentMethodID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	paymentMethod, err := s.dal.PaymentMethod(ctx, paymentMethodID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "PaymentMethod %s Not Found", paymentMethodID)
	} else if err != nil {
		return nil, grpcError(err)
	}
	if err := s.deletePaymentMethod(ctx, paymentMethod, s.dal); err != nil {
		return nil, grpcError(err)
	}
	resp, err := s.PaymentMethods(ctx, &payments.PaymentMethodsRequest{EntityID: paymentMethod.EntityID})
	if err != nil {
		return nil, grpcError(err)
	}
	return &payments.DeletePaymentMethodResponse{
		PaymentMethods: resp.PaymentMethods,
	}, nil
}

func (s *server) deletePaymentMethod(ctx context.Context, paymentMethod *dal.PaymentMethod, dl dal.DAL) error {
	if err := dl.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		vendorAccount, err := dl.VendorAccount(ctx, paymentMethod.VendorAccountID)
		if err != nil {
			return errors.Trace(err)
		}
		customer, err := dl.Customer(ctx, paymentMethod.CustomerID)
		if err != nil {
			return errors.Trace(err)
		}
		if _, err := dl.DeletePaymentMethod(ctx, paymentMethod.ID); err != nil {
			return errors.Trace(err)
		}
		switch paymentMethod.StorageType {
		case dal.PaymentMethodStorageTypeStripe:
			// TODO: This should be an inner switch on the type (CARD etc, need to store that in the record)
			if err := s.stripeClient.DeleteCard(ctx, paymentMethod.StorageID, &stripe.CardParams{
				Customer: customer.StorageID,
				Params: stripe.Params{
					StripeAccount: vendorAccount.ConnectedAccountID,
				},
			}); err != nil {
				if istripe.ErrCode(errors.Cause(err)) == stripe.Missing {
					golog.Infof("Attempted to delete card %s mapped to payment method %s but Stripe reported it missing already. Moving on.", paymentMethod.StorageID, paymentMethod.ID)
				} else {
					return errors.Trace(err)
				}
			}
		default:
			return errors.Errorf("Unhandled payment method storage type %s for %s in deletion", paymentMethod.StorageType, paymentMethod.ID)
		}
		// If this is the master account, cleanup the card from sub vendors
		if vendorAccount.ID == s.masterVendorAccount.ID {
			// TODO: Tracking these payment method groupings by fingerprint locks us into only supporting types that provide a fingerprint.
			//	Should consider a groping id for future payment types.
			paymentMethods, err := dl.PaymentMethodsWithFingerprint(ctx, paymentMethod.StorageFingerprint)
			if err != nil {
				return errors.Trace(err)
			}
			for _, pm := range paymentMethods {
				if err := s.deletePaymentMethod(ctx, pm, dl); err != nil {
					return errors.Trace(err)
				}
			}
		}
		return nil
	}); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (s *server) PaymentMethods(ctx context.Context, req *payments.PaymentMethodsRequest) (*payments.PaymentMethodsResponse, error) {
	if req.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "EntityID required")
	}
	paymentMethods, err := s.dal.EntityPaymentMethods(ctx, s.masterVendorAccount.ID, req.EntityID)
	if err != nil {
		return nil, grpcError(err)
	}
	customer, err := s.dal.CustomerForVendor(ctx, s.masterVendorAccount.ID, req.EntityID)
	if err != nil {
		return nil, grpcError(err)
	}
	rPaymentMethods, err := transformPaymentMethodsToResponse(ctx, customer, paymentMethods, s.stripeClient)
	if err != nil {
		return nil, grpcError(err)
	}
	return &payments.PaymentMethodsResponse{
		PaymentMethods: rPaymentMethods,
	}, nil
}

func (s *server) UpdateVendorAccount(ctx context.Context, req *payments.UpdateVendorAccountRequest) (*payments.UpdateVendorAccountResponse, error) {
	if req.VendorAccountID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "VendorAccountID required")
	}
	vendorAccountID, err := dal.ParseVendorAccountID(req.VendorAccountID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	lifecycle, err := transformVendorAccountLifecycleToDAL(req.Lifecycle)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	changeState, err := transformVendorAccountChangeStateToDAL(req.ChangeState)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		vendorAccount, err := dl.VendorAccount(ctx, vendorAccountID, dal.ForUpdate)
		if errors.Cause(err) == dal.ErrNotFound {
			return grpcErrorf(codes.NotFound, "Vendor Account %s Not Found", vendorAccountID)
		} else if err != nil {
			return grpcError(err)
		}
		// If we're already there do nothing
		if vendorAccount.Lifecycle == lifecycle && vendorAccount.ChangeState == changeState {
			return nil
		}
		if err := dl.UpdateVendorAccount(ctx, vendorAccountID, &dal.VendorAccountUpdate{
			Lifecycle:   lifecycle,
			ChangeState: changeState,
		}); err != nil {
			return grpcError(err)
		}
		return nil
	}); err != nil {
		return nil, grpcError(err)
	}
	return &payments.UpdateVendorAccountResponse{}, nil
}

func (s *server) VendorAccounts(ctx context.Context, req *payments.VendorAccountsRequest) (*payments.VendorAccountsResponse, error) {
	if req.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "EntityID required")
	}
	vendorAccounts, err := s.dal.EntityVendorAccounts(ctx, req.EntityID)
	if err != nil {
		return nil, grpcError(err)
	}
	return &payments.VendorAccountsResponse{
		VendorAccounts: transformVendorAccountsToResponse(vendorAccounts),
	}, nil
}
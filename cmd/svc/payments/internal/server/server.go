package server

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	istripe "github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/smet"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
	stripe "github.com/stripe/stripe-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type server struct {
	dal             dal.DAL
	directoryClient directory.DirectoryClient
	settingsClient  settings.SettingsClient
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
		golog.Errorf("Encountered an error when validating Stripe credentials: %s", err)
	}
	golog.Infof("Master Stripe Account: %+v", masterStripeAccount)
	return masterVendorAccount, nil
}

func (s *server) AcceptPayment(ctx context.Context, req *payments.AcceptPaymentRequest) (*payments.AcceptPaymentResponse, error) {
	if req.PaymentID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "PaymentID required")
	}
	paymentID, err := dal.ParsePaymentID(req.PaymentID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	if req.PaymentMethodID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "PaymentMethodID required")
	}
	paymentMethodID, err := dal.ParsePaymentMethodID(req.PaymentMethodID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}

	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		// Lock the row we intend to manipulate
		payment, err := dl.Payment(ctx, paymentID, dal.ForUpdate)
		if errors.Cause(err) == dal.ErrNotFound {
			return grpc.Errorf(codes.NotFound, "Payment %s Not Found", paymentID)
		} else if err != nil {
			return errors.Trace(err)
		}
		paymentMethod, err := dl.PaymentMethod(ctx, paymentMethodID)
		if errors.Cause(err) == dal.ErrNotFound {
			return grpc.Errorf(codes.NotFound, "PaymentMethod %s Not Found", paymentID)
		} else if err != nil {
			return errors.Trace(err)
		}

		// Set the default payment method to the one we're accepting this payment with
		if err := s.setDefaultPaymentMethod(ctx, dl, paymentMethod.ID, paymentMethod.EntityID); err != nil {
			return errors.Trace(err)
		}

		// If nothing is changing move on
		if payment.ChangeState == dal.PaymentChangeStateNone &&
			payment.Lifecycle == dal.PaymentLifecycleAccepted &&
			payment.PaymentMethodID == paymentMethod.ID {
			golog.Infof("Payment %s is already in the accepted state with payment method %s ignoring double accept", payment.ID, paymentMethod.ID)
			return nil
		}

		// Acceptable States For Update
		// 1. If we are just changing the payment method
		// 2. We are accepting for the first time (NONE, SUBMITTED)
		if (payment.ChangeState == dal.PaymentChangeStateNone && payment.Lifecycle == dal.PaymentLifecycleAccepted) ||
			(payment.ChangeState == dal.PaymentChangeStateNone && payment.Lifecycle == dal.PaymentLifecycleSubmitted) {
			if _, err := dl.UpdatePayment(ctx, paymentID, &dal.PaymentUpdate{
				ChangeState:     dal.PaymentChangeStateNone,
				Lifecycle:       dal.PaymentLifecycleAccepted,
				PaymentMethodID: &paymentMethod.ID,
			}); err != nil {
				return errors.Trace(err)
			}
			smet.GetCounter("Server-AcceptPayment-Success").Inc()
		} else {
			golog.Infof("Payment %s is in state %s|%s - it cannot be accepted - ignoring accept", payment.ID, payment.ChangeState, payment.Lifecycle)
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}

	resp, err := s.Payment(ctx, &payments.PaymentRequest{PaymentID: req.PaymentID})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &payments.AcceptPaymentResponse{
		Payment: resp.Payment,
	}, nil
}

func (s *server) CreatePayment(ctx context.Context, req *payments.CreatePaymentRequest) (*payments.CreatePaymentResponse, error) {
	if req.RequestingEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "RequestingEntityID required")
	}
	if req.Amount <= 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "Positive no zero Amount required")
	}
	if req.Currency == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Currency required")
	}
	vendorAccounts, err := s.dal.EntityVendorAccounts(ctx, req.RequestingEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(vendorAccounts) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "Vendor Account for %s Not Found", req.RequestingEntityID)
	}
	// For now just assume there will be only 1
	vendorAccount := vendorAccounts[0]

	paymentID, err := s.dal.InsertPayment(ctx, &dal.Payment{
		VendorAccountID: vendorAccount.ID,
		Currency:        req.Currency,
		Amount:          req.Amount,
		ChangeState:     dal.PaymentChangeStatePending,
		Lifecycle:       dal.PaymentLifecycleSubmitted,
	})

	resp, err := s.Payment(ctx, &payments.PaymentRequest{PaymentID: paymentID.String()})
	if err != nil {
		return nil, errors.Trace(err)
	}
	smet.GetCounter("Server-CreatePayment-Success").Inc()
	return &payments.CreatePaymentResponse{
		Payment: resp.Payment,
	}, nil
}

func (s *server) CreatePaymentMethod(ctx context.Context, req *payments.CreatePaymentMethodRequest) (*payments.CreatePaymentMethodResponse, error) {
	if req.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID required")
	}
	customer, err := AddCustomer(ctx, s.masterVendorAccount, req.EntityID, s.dal, s.directoryClient, s.stripeClient)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var token string
	switch req.StorageType {
	case payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE:
		switch req.Type {
		case payments.PAYMENT_METHOD_TYPE_CARD:
			stripeCard := req.GetStripeCard()
			if stripeCard.Token == "" {
				return nil, grpc.Errorf(codes.InvalidArgument, "Token required")
			}
			token = stripeCard.Token
		default:
			return nil, errors.Errorf("Unhandled payment method type %s for customer %s payment method addition", req.Type, customer.ID)
		}
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unhandled payment method storage type %s", req.StorageType)
	}
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		paymentMethod, err := AddPaymentMethod(ctx, s.masterVendorAccount, customer, req.Type, &LiteralTokenSource{T: token}, dl, s.stripeClient)
		if err != nil {
			return errors.Trace(err)
		}
		if err := s.setDefaultPaymentMethod(ctx, dl, paymentMethod.ID, req.EntityID); err != nil {
			return errors.Trace(err)
		}
		return nil
	}); err != nil {
		if IsPaymentMethodError(errors.Cause(err)) {
			return nil, grpc.Errorf(payments.PaymentMethodError, PaymentMethodErrorMesssage(errors.Cause(err)))
		}
		return nil, errors.Trace(err)
	}
	resp, err := s.PaymentMethods(ctx, &payments.PaymentMethodsRequest{EntityID: req.EntityID})
	if err != nil {
		return nil, errors.Trace(err)
	}
	smet.GetCounter("Server-CreatePaymentMethod-Success").Inc()
	return &payments.CreatePaymentMethodResponse{
		PaymentMethods: resp.PaymentMethods,
	}, nil
}

// AddCustomer adds a customer to the provided vendor account
func AddCustomer(
	ctx context.Context,
	vendorAccount *dal.VendorAccount,
	entityID string,
	dl dal.DAL,
	directoryClient directory.DirectoryClient,
	stripeClient istripe.IdempotentStripeClient) (*dal.Customer, error) {

	// Check to see if we've already added this customer
	customer, err := dl.CustomerForVendor(ctx, vendorAccount.ID, entityID)
	if err != nil && errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	} else if customer != nil {
		golog.Debugf("Customer FOUND - Entity: %s for VendorAccount: %s NOT ADDING", entityID, vendorAccount.ID)
		return customer, nil
	}
	//TODO: Assert remote existance

	var newCustomer *dal.Customer
	// This transaction is likely ignored since an outer one is in effect, but do it anyways
	if err := dl.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		switch vendorAccount.AccountType {
		case dal.VendorAccountAccountTypeStripe:
			stripeCustomer, err := stripeClient.CreateCustomer(ctx, &stripe.CustomerParams{
				Params: stripe.Params{
					StripeAccount: vendorAccount.ConnectedAccountID,
					Meta:          map[string]string{"entity_id": entityID},
				},
			})
			if err != nil {
				return errors.Trace(err)
			}
			newCustomer = &dal.Customer{
				StorageType: dal.CustomerStorageTypeStripe,
				StorageID:   stripeCustomer.ID,
			}
		default:
			return errors.Errorf("Unknown vendor account type %s for vendor account %s customer creation", vendorAccount.AccountType, vendorAccount.ID)
		}
		// sanity
		if newCustomer == nil {
			return errors.Errorf("nil newCustomer, this should never happen")
		}
		newCustomer.VendorAccountID = vendorAccount.ID
		newCustomer.EntityID = entityID
		newCustomer.Lifecycle = dal.CustomerLifecycleActive
		newCustomer.ChangeState = dal.CustomerChangeStateNone

		id, err := dl.InsertCustomer(ctx, newCustomer)
		if err != nil {
			return errors.Trace(err)
		}
		newCustomer.ID = id
		golog.Debugf("Customer NOT FOUND - Entity: %s for VendorAccount: %s, ADDED - %+v", entityID, vendorAccount.ID, newCustomer)
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return newCustomer, nil
}

type TokenSource interface {
	Token() (string, error)
}

type LiteralTokenSource struct {
	T string
}

func (lt *LiteralTokenSource) Token() (string, error) {
	return lt.T, nil
}

type DynamicTokenSource struct {
	D func() (string, error)
}

func (dt *DynamicTokenSource) Token() (string, error) {
	return dt.D()
}

// AddPaymentMethod adds a payment method to the provided vendor account and customer
func AddPaymentMethod(
	ctx context.Context,
	vendorAccount *dal.VendorAccount,
	customer *dal.Customer,
	paymentMethodType payments.PaymentMethodType,
	tokenSource TokenSource,
	dl dal.DAL,
	stripeClient istripe.IdempotentStripeClient) (*dal.PaymentMethod, error) {
	token, err := tokenSource.Token()
	if err != nil {
		return nil, errors.Errorf("Error getting token for payment method addition: %s", err)
	}
	var newPaymentMethod *dal.PaymentMethod
	switch vendorAccount.AccountType {
	case dal.VendorAccountAccountTypeStripe:
		switch paymentMethodType {
		case payments.PAYMENT_METHOD_TYPE_CARD:
			stripeCard, err := stripeClient.CreateCard(ctx, &stripe.CardParams{
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
				Type:               dal.PaymentMethodTypeCard,
				StorageType:        dal.PaymentMethodStorageTypeStripe,
				StorageID:          stripeCard.ID,
				StorageFingerprint: stripeCard.Fingerprint,
				Brand:              string(stripeCard.Brand),
				Last4:              stripeCard.LastFour,
				ExpMonth:           int(stripeCard.Month),
				ExpYear:            int(stripeCard.Year),
				TokenizationMethod: string(stripeCard.TokenizationMethod),
			}
		default:
			return nil, errors.Errorf("Unhandled payment method type %s for vendor account %s payment method addition", paymentMethodType, vendorAccount.ID)
		}
	default:
		return nil, errors.Errorf("Unhandled vendor account type %s for vendor account %s payment method addition", vendorAccount.AccountType, vendorAccount.ID)
	}
	// sanity
	if newPaymentMethod == nil {
		return nil, errors.Errorf("nil newPaymentMethod, this should never happen")
	}
	newPaymentMethod.VendorAccountID = vendorAccount.ID
	newPaymentMethod.CustomerID = customer.ID
	newPaymentMethod.EntityID = customer.EntityID
	newPaymentMethod.Lifecycle = dal.PaymentMethodLifecycleActive
	newPaymentMethod.ChangeState = dal.PaymentMethodChangeStateNone

	// Check to see if we've already added this payment method - the stripe endpoint is idempotent
	existingPaymentMethod, err := dl.PaymentMethodWithFingerprint(ctx, customer.ID, newPaymentMethod.StorageFingerprint, newPaymentMethod.TokenizationMethod)
	if err != nil && errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	} else if existingPaymentMethod != nil {
		golog.Debugf("Payment Method FOUND - Fingerprint: %s Entity: %s for VendorAccount: %s - NOT ADDING", existingPaymentMethod.StorageFingerprint, existingPaymentMethod.EntityID, vendorAccount.ID)
		if existingPaymentMethod.Lifecycle == dal.PaymentMethodLifecycleDeleted {
			golog.Debugf("Payment Method added, but was previously deleted - resurrecting record")
			if _, err := dl.UpdatePaymentMethod(ctx, existingPaymentMethod.ID, &dal.PaymentMethodUpdate{
				Lifecycle:   dal.PaymentMethodLifecycleActive,
				ChangeState: dal.PaymentMethodChangeStateNone,
				StorageID:   &newPaymentMethod.StorageID,
			}); err != nil {
				return nil, errors.Trace(err)
			}
			existingPaymentMethod.Lifecycle = dal.PaymentMethodLifecycleActive
			existingPaymentMethod.ChangeState = dal.PaymentMethodChangeStateNone
			existingPaymentMethod.StorageID = newPaymentMethod.StorageID
		}
		return existingPaymentMethod, nil
	}
	golog.Debugf("Payment Method NOT FOUND - Fingerprint: %s Entity: %s for VendorAccount: %s - ADDING", newPaymentMethod.StorageFingerprint, newPaymentMethod.EntityID, vendorAccount.ID)
	id, err := dl.InsertPaymentMethod(ctx, newPaymentMethod)
	if err != nil {
		return nil, errors.Trace(err)
	}
	newPaymentMethod.ID = id
	return newPaymentMethod, nil
}

// TODO: Dedupe inserts on account ID in the event of multiple connections from same account
func (s *server) ConnectVendorAccount(ctx context.Context, req *payments.ConnectVendorAccountRequest) (*payments.ConnectVendorAccountResponse, error) {
	if req.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID required")
	}
	var vendorAccount *dal.VendorAccount
	switch req.Type {
	case payments.VENDOR_ACCOUNT_TYPE_STRIPE:
		stripeReq := req.GetStripeRequest()
		if stripeReq.Code == "" {
			return nil, grpc.Errorf(codes.InvalidArgument, "Code required")
		}
		accessTokenResponse, err := s.stripeOAuth.RequestStripeAccessToken(stripeReq.Code)
		if err != nil {
			return nil, errors.Trace(err)
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Unsupported vendor account type %s", req.Type)
	}
	// sanity
	if vendorAccount == nil {
		return nil, errors.Errorf("nil vendorAccount, this should never happen")
	}
	vendorAccount.Lifecycle = dal.VendorAccountLifecycleConnected
	vendorAccount.ChangeState = dal.VendorAccountChangeStateNone
	vendorAccount.EntityID = req.EntityID
	if _, err := s.dal.InsertVendorAccount(ctx, vendorAccount); err != nil {
		return nil, errors.Trace(err)
	}

	// Look up the new set of vendor accounts associated with the entity ID now
	vendorAccounts, err := s.VendorAccounts(ctx, &payments.VendorAccountsRequest{EntityID: req.EntityID})
	if err != nil {
		return nil, err
	}
	smet.GetCounter("Server-ConnectVendorAccount-Success").Inc()
	return &payments.ConnectVendorAccountResponse{
		VendorAccounts: vendorAccounts.VendorAccounts,
	}, nil
}

// TODO: This is perhaps something we could perhaps so more lazily with setting the record into PENDING/DELETING and having a worker clean it up.
// 	While I like that solution better, there are race conditions around deleting and readding the same card before the worker runs to consider.
//	Leave this synchronous for now.
func (s *server) DeletePaymentMethod(ctx context.Context, req *payments.DeletePaymentMethodRequest) (*payments.DeletePaymentMethodResponse, error) {
	if req.PaymentMethodID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "PaymentMethodID required")
	}
	paymentMethodID, err := dal.ParsePaymentMethodID(req.PaymentMethodID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid PaymentMethodID")
	}
	paymentMethod, err := s.dal.PaymentMethod(ctx, paymentMethodID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "PaymentMethod %s Not Found", paymentMethodID)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		if err := s.deletePaymentMethod(ctx, paymentMethod.ID, dl); err != nil {
			return errors.Trace(err)
		}
		if paymentMethod.Default {
			// Payment methods are returned sorted by order created desc, so we should set the first payment method returned as the default
			paymentMethods, err := dl.EntityPaymentMethods(ctx, paymentMethod.VendorAccountID, paymentMethod.EntityID)
			if err != nil {
				return errors.Trace(err)
			}
			if len(paymentMethods) != 0 {
				if err := s.setDefaultPaymentMethod(ctx, dl, paymentMethods[0].ID, paymentMethod.EntityID); err != nil {
					return errors.Trace(err)
				}
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	resp, err := s.PaymentMethods(ctx, &payments.PaymentMethodsRequest{EntityID: paymentMethod.EntityID})
	if err != nil {
		return nil, errors.Trace(err)
	}
	smet.GetCounter("Server-DeletePaymentMethod-Success").Inc()
	return &payments.DeletePaymentMethodResponse{
		PaymentMethods: resp.PaymentMethods,
	}, nil
}

func (s *server) deletePaymentMethod(ctx context.Context, paymentMethodID dal.PaymentMethodID, dl dal.DAL) error {
	if err := dl.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		// Lock the row for update -- This double read is inefficient, ignore for now
		pm, err := dl.PaymentMethod(ctx, paymentMethodID, dal.ForUpdate)
		if err != nil {
			return errors.Trace(err)
		}
		vendorAccount, err := dl.VendorAccount(ctx, pm.VendorAccountID)
		if err != nil {
			return errors.Trace(err)
		}
		customer, err := dl.Customer(ctx, pm.CustomerID)
		if err != nil {
			return errors.Trace(err)
		}
		if _, err := dl.UpdatePaymentMethod(ctx, pm.ID, &dal.PaymentMethodUpdate{
			Lifecycle:   dal.PaymentMethodLifecycleDeleted,
			ChangeState: dal.PaymentMethodChangeStateNone,
		}); err != nil {
			return errors.Trace(err)
		}
		switch pm.StorageType {
		case dal.PaymentMethodStorageTypeStripe:
			// TODO: This should be an inner switch on the type (CARD etc, need to store that in the record)
			if err := s.stripeClient.DeleteCard(ctx, pm.StorageID, &stripe.CardParams{
				Customer: customer.StorageID,
				Params: stripe.Params{
					StripeAccount: vendorAccount.ConnectedAccountID,
				},
			}); err != nil {
				if istripe.ErrCode(errors.Cause(err)) == stripe.Missing {
					golog.Infof("Attempted to delete card %s mapped to payment method %s but Stripe reported it missing already. Moving on.", pm.StorageID, pm.ID)
				} else {
					return errors.Trace(err)
				}
			}
		default:
			return errors.Errorf("Unhandled payment method storage type %s for %s in deletion", pm.StorageType, pm.ID)
		}
		return nil
	}); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (s *server) Payment(ctx context.Context, req *payments.PaymentRequest) (*payments.PaymentResponse, error) {
	if req.PaymentID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "PaymentID required")
	}
	paymentID, err := dal.ParsePaymentID(req.PaymentID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid PaymentID")
	}
	payment, err := s.dal.Payment(ctx, paymentID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Payment %s Not Found", paymentID)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	rPayment, err := transformPaymentToResponse(ctx, payment, s.dal, s.stripeClient)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &payments.PaymentResponse{
		Payment: rPayment,
	}, nil
}

func (s *server) SubmitPayment(ctx context.Context, req *payments.SubmitPaymentRequest) (*payments.SubmitPaymentResponse, error) {
	if req.PaymentID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "PaymentID required")
	}
	paymentID, err := dal.ParsePaymentID(req.PaymentID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid PaymentID")
	}
	// Payments don't have to be associated with a thread_id, but if it is track it.
	var threadID *string
	if req.ThreadID != "" {
		threadID = &req.ThreadID
	}

	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		// Lock the row we intend to manipulate
		payment, err := dl.Payment(ctx, paymentID, dal.ForUpdate)
		if errors.Cause(err) == dal.ErrNotFound {
			return grpc.Errorf(codes.NotFound, "Payment %s Not Found", paymentID)
		} else if err != nil {
			return errors.Trace(err)
		}
		if payment.ChangeState == dal.PaymentChangeStateNone && payment.Lifecycle == dal.PaymentLifecycleSubmitted {
			golog.Infof("Payment %s is already in the submitted state %s|%s ignoring double submit", paymentID, payment.ChangeState, payment.Lifecycle)
			return nil
		}
		if payment.ChangeState == dal.PaymentChangeStatePending && payment.Lifecycle != dal.PaymentLifecycleSubmitted {
			return grpc.Errorf(codes.FailedPrecondition, "Payment %s is in state %s|%s - it cannot be submitted", payment.ID, payment.ChangeState, payment.Lifecycle)
		}
		if _, err := dl.UpdatePayment(ctx, paymentID, &dal.PaymentUpdate{
			ChangeState: dal.PaymentChangeStateNone,
			Lifecycle:   dal.PaymentLifecycleSubmitted,
			ThreadID:    threadID,
		}); err != nil {
			return errors.Trace(err)
		}
		smet.GetCounter("Server-SubmitPayment-Success").Inc()
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}

	resp, err := s.Payment(ctx, &payments.PaymentRequest{PaymentID: req.PaymentID})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &payments.SubmitPaymentResponse{
		Payment: resp.Payment,
	}, nil
}

func (s *server) PaymentMethod(ctx context.Context, req *payments.PaymentMethodRequest) (*payments.PaymentMethodResponse, error) {
	if req.PaymentMethodID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "PaymentMethodID required")
	}
	paymentMethodID, err := dal.ParsePaymentMethodID(req.PaymentMethodID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid PaymentMethodID")
	}
	paymentMethod, err := s.dal.PaymentMethod(ctx, paymentMethodID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "PaymentMethod %s Not Found", req.PaymentMethodID)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &payments.PaymentMethodResponse{
		PaymentMethod: transformPaymentMethodToResponse(paymentMethod),
	}, nil
}

func (s *server) PaymentMethods(ctx context.Context, req *payments.PaymentMethodsRequest) (*payments.PaymentMethodsResponse, error) {
	if req.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID required")
	}
	paymentMethods, err := s.dal.EntityPaymentMethods(ctx, s.masterVendorAccount.ID, req.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid EntityID")
	}
	return &payments.PaymentMethodsResponse{
		PaymentMethods: transformPaymentMethodsToResponse(paymentMethods),
	}, nil
}

func (s *server) UpdateVendorAccount(ctx context.Context, req *payments.UpdateVendorAccountRequest) (*payments.UpdateVendorAccountResponse, error) {
	if req.VendorAccountID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "VendorAccountID required")
	}
	vendorAccountID, err := dal.ParseVendorAccountID(req.VendorAccountID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	lifecycle, err := transformVendorAccountLifecycleToDAL(req.Lifecycle)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	changeState, err := transformVendorAccountChangeStateToDAL(req.ChangeState)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		vendorAccount, err := dl.VendorAccount(ctx, vendorAccountID, dal.ForUpdate)
		if errors.Cause(err) == dal.ErrNotFound {
			return grpc.Errorf(codes.NotFound, "Vendor Account %s Not Found", vendorAccountID)
		} else if err != nil {
			return errors.Trace(err)
		}
		// If we're already there do nothing
		if vendorAccount.Lifecycle == lifecycle && vendorAccount.ChangeState == changeState {
			return nil
		}
		if err := dl.UpdateVendorAccount(ctx, vendorAccountID, &dal.VendorAccountUpdate{
			Lifecycle:   lifecycle,
			ChangeState: changeState,
		}); err != nil {
			return errors.Trace(err)
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &payments.UpdateVendorAccountResponse{}, nil
}

func (s *server) VendorAccounts(ctx context.Context, req *payments.VendorAccountsRequest) (*payments.VendorAccountsResponse, error) {
	if req.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID required")
	}
	vendorAccounts, err := s.dal.EntityVendorAccounts(ctx, req.EntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &payments.VendorAccountsResponse{
		VendorAccounts: transformVendorAccountsToResponse(vendorAccounts),
	}, nil
}

func (s *server) setDefaultPaymentMethod(ctx context.Context, dl dal.DAL, paymentMethodID dal.PaymentMethodID, entityID string) error {
	err := dl.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		// TODO: Could perhaps optimize this read out.
		paymentMethod, err := dl.PaymentMethod(ctx, paymentMethodID)
		if err != nil {
			return errors.Trace(err)
		}
		if paymentMethod.EntityID != entityID {
			return errors.Errorf("Entity %s does not own payment method %s - cannot set as default", entityID, paymentMethodID)
		}
		if paymentMethod.VendorAccountID != s.masterVendorAccount.ID {
			return errors.Errorf("Payment method %s is not owned by the master account - cannot set as default", paymentMethodID)
		}
		paymentMethods, err := dl.EntityPaymentMethods(ctx, s.masterVendorAccount.ID, entityID, dal.ForUpdate)
		if err != nil {
			return errors.Trace(err)
		}
		for _, pm := range paymentMethods {
			if pm.Default {
				if _, err := dl.UpdatePaymentMethod(ctx, paymentMethodID, &dal.PaymentMethodUpdate{
					Lifecycle:   pm.Lifecycle,
					ChangeState: pm.ChangeState,
					Default:     ptr.Bool(false),
				}); err != nil {
					return errors.Trace(err)
				}
			}
		}
		_, err = dl.UpdatePaymentMethod(ctx, paymentMethodID, &dal.PaymentMethodUpdate{
			Lifecycle:   paymentMethod.Lifecycle,
			ChangeState: paymentMethod.ChangeState,
			Default:     ptr.Bool(true),
		})
		return errors.Trace(err)
	})
	return errors.Trace(err)
}

package workers

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/testutil"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"

	"github.com/stripe/stripe-go"
)

func TestPaymentNoneAccepted(t *testing.T) {
	dmock := testutil.NewMockDAL(t)
	defer dmock.Finish()
	smock := testutil.NewMockIdempotentStripeClient(t)
	defer smock.Finish()
	directorymock := dirmock.New(t)
	defer directorymock.Finish()

	paymentID, err := dal.NewPaymentID()
	test.OK(t, err)
	vendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	masterVendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	masterPaymentMethodID, err := dal.NewPaymentMethodID()
	test.OK(t, err)
	masterCustomerID, err := dal.NewCustomerID()
	test.OK(t, err)
	customerID, err := dal.NewCustomerID()
	test.OK(t, err)

	// look for payments to processs
	dmock.Expect(mock.NewExpectation(
		dmock.PaymentsInState,
		dal.PaymentLifecycleAccepted,
		dal.PaymentChangeStateNone,
		int64(10),
		[]dal.QueryOption{dal.ForUpdate}).WithReturns([]*dal.Payment{
		{
			ID:              paymentID,
			VendorAccountID: vendorAccountID,
			PaymentMethodID: masterPaymentMethodID,
		},
	}, nil))

	// lookup vendored account that the payment method method is linked to
	dmock.Expect(mock.NewExpectation(dmock.VendorAccount, vendorAccountID).WithReturns(&dal.VendorAccount{
		ID:                 vendorAccountID,
		AccountType:        dal.VendorAccountAccountTypeStripe,
		ConnectedAccountID: "stripeConnectedAccountID",
	}, nil))

	// lookup the payment method (which is currently linked to the masterVendorAccount rather than the customer vendor account)
	dmock.Expect(mock.NewExpectation(dmock.PaymentMethod, masterPaymentMethodID, []dal.QueryOption(nil)).WithReturns(&dal.PaymentMethod{
		ID:                 masterPaymentMethodID,
		EntityID:           "entityID",
		VendorAccountID:    masterVendorAccountID,
		StorageFingerprint: "stripeFingerprint",
		TokenizationMethod: "token",
		CustomerID:         masterCustomerID,
		StorageID:          "masterStripeCardStorageID",
		Type:               dal.PaymentMethodTypeCard,
	}, nil))

	// look up an existing customer for the vendor which should come back as non-existent
	dmock.Expect(mock.NewExpectation(dmock.CustomerForVendor, vendorAccountID, "entityID", []dal.QueryOption(nil)).WithReturns((*dal.Customer)(nil), dal.ErrNotFound))
	dmock.Expect(mock.NewExpectation(dmock.CustomerForVendor, vendorAccountID, "entityID", []dal.QueryOption(nil)).WithReturns((*dal.Customer)(nil), dal.ErrNotFound))

	// create the customer given that it doesn't exist for the vendor
	dmock.Expect(mock.NewExpectation(dmock.InsertCustomer, &dal.Customer{
		StorageType:     dal.CustomerStorageTypeStripe,
		StorageID:       "stripeCustomerID",
		VendorAccountID: vendorAccountID,
		EntityID:        "entityID",
		Lifecycle:       dal.CustomerLifecycleActive,
		ChangeState:     dal.CustomerChangeStateNone,
	}).WithReturns(customerID, nil))

	// lookup the card based on the fingerprint for the customer associated with the vendor
	dmock.Expect(mock.NewExpectation(
		dmock.PaymentMethodWithFingerprint,
		customerID,
		"stripeFingerprint", "token",
		[]dal.QueryOption(nil)).WithReturns((*dal.PaymentMethod)(nil), dal.ErrNotFound))

	// lookup the master customer
	dmock.Expect(mock.NewExpectation(dmock.Customer, masterCustomerID, []dal.QueryOption(nil)).WithReturns(&dal.Customer{
		StorageID: "masterCustomerStripeID",
	}, nil))

	// lookup again to ensure that card is not added
	dmock.Expect(mock.NewExpectation(
		dmock.PaymentMethodWithFingerprint,
		customerID,
		"newCardFingerprint", "newCardToken",
		[]dal.QueryOption(nil)).WithReturns((*dal.PaymentMethod)(nil), dal.ErrNotFound))

	paymentMethodID, err := dal.NewPaymentMethodID()
	test.OK(t, err)

	// create new card for customer
	dmock.Expect(mock.NewExpectation(dmock.InsertPaymentMethod, &dal.PaymentMethod{
		Type:               dal.PaymentMethodTypeCard,
		StorageType:        dal.PaymentMethodStorageTypeStripe,
		StorageID:          "stripeCardID",
		StorageFingerprint: "newCardFingerprint",
		Brand:              "Visa",
		Last4:              "1234",
		ExpMonth:           int(10),
		ExpYear:            int(24),
		TokenizationMethod: "newCardToken",
		VendorAccountID:    vendorAccountID,
		CustomerID:         customerID,
		EntityID:           "entityID",
		Lifecycle:          dal.PaymentMethodLifecycleActive,
		ChangeState:        dal.PaymentMethodChangeStateNone,
	}).WithReturns(paymentMethodID, nil))

	// map the existing payment method to the newly created payment method
	dmock.Expect(mock.NewExpectation(dmock.UpdatePaymentMethod, paymentID, &dal.PaymentUpdate{
		Lifecycle:       dal.PaymentLifecycleAccepted,
		ChangeState:     dal.PaymentChangeStateNone,
		PaymentMethodID: &paymentMethodID,
	}))

	// Move payment on to the next phase
	dmock.Expect(mock.NewExpectation(dmock.UpdatePayment, paymentID, &dal.PaymentUpdate{
		Lifecycle:   dal.PaymentLifecycleProcessing,
		ChangeState: dal.PaymentChangeStatePending,
	}))

	smock.Expect(mock.NewExpectation(smock.CreateCustomer, &stripe.CustomerParams{
		Params: stripe.Params{
			StripeAccount: "stripeConnectedAccountID",
			Meta:          map[string]string{"entity_id": "entityID"},
		},
	}).WithReturns(&stripe.Customer{
		ID: "stripeCustomerID",
	}, nil))

	smock.Expect(mock.NewExpectation(smock.Token, &stripe.TokenParams{
		Customer: "masterCustomerStripeID",
		Card:     &stripe.CardParams{Token: "masterStripeCardStorageID"},
		Params: stripe.Params{
			StripeAccount: "stripeConnectedAccountID",
		},
	}).WithReturns(&stripe.Token{
		ID: "token",
	}, nil))

	smock.Expect(mock.NewExpectation(smock.CreateCard, &stripe.CardParams{
		Customer: "stripeCustomerID",
		Token:    "token",
		Params: stripe.Params{
			StripeAccount: "stripeConnectedAccountID",
			Meta: map[string]string{
				"entity_id":   "entityID",
				"customer_id": customerID.String(),
			},
		},
	}).WithReturns(&stripe.Card{
		ID:                 "stripeCardID",
		Fingerprint:        "newCardFingerprint",
		TokenizationMethod: stripe.TokenizationMethod("newCardToken"),
		Brand:              stripe.CardBrand("Visa"),
		Month:              uint8(10),
		Year:               uint16(24),
		LastFour:           "1234",
	}, nil))

	directorymock.Expect(mock.NewExpectation(directorymock.CreateExternalLink, &directory.CreateExternalLinkRequest{
		EntityID: "entityID",
		Name:     "Stripe",
		URL:      "https://dashboard.stripe.com/test/customers/stripeCustomerID",
	}))

	directorymock.Expect(mock.NewExpectation(directorymock.CreateExternalIDs, &directory.CreateExternalIDsRequest{
		EntityID:    "entityID",
		ExternalIDs: []string{"stripe_stripeCustomerID"},
	}))

	w := New(dmock, directorymock, nil, smock, "", "", "")
	w.processPaymentNoneAccepted()
}

func TestPaymentNoneAccepted_Idempotent(t *testing.T) {
	dmock := testutil.NewMockDAL(t)
	defer dmock.Finish()
	smock := testutil.NewMockIdempotentStripeClient(t)
	defer smock.Finish()
	directorymock := dirmock.New(t)
	defer directorymock.Finish()

	paymentID, err := dal.NewPaymentID()
	test.OK(t, err)
	vendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	customerID, err := dal.NewCustomerID()
	test.OK(t, err)
	paymentMethodID, err := dal.NewPaymentMethodID()
	test.OK(t, err)

	// look for payments to processs
	dmock.Expect(mock.NewExpectation(
		dmock.PaymentsInState,
		dal.PaymentLifecycleAccepted,
		dal.PaymentChangeStateNone,
		int64(10),
		[]dal.QueryOption{dal.ForUpdate}).WithReturns([]*dal.Payment{
		{
			ID:              paymentID,
			VendorAccountID: vendorAccountID,
			PaymentMethodID: paymentMethodID,
		},
	}, nil))

	// lookup vendored account that the payment method method is linked to
	dmock.Expect(mock.NewExpectation(dmock.VendorAccount, vendorAccountID).WithReturns(&dal.VendorAccount{
		ID:                 vendorAccountID,
		AccountType:        dal.VendorAccountAccountTypeStripe,
		ConnectedAccountID: "stripeConnectedAccountID",
	}, nil))

	dmock.Expect(mock.NewExpectation(dmock.PaymentMethod, paymentMethodID, []dal.QueryOption(nil)).WithReturns(&dal.PaymentMethod{
		ID:                 paymentMethodID,
		EntityID:           "entityID",
		VendorAccountID:    vendorAccountID,
		StorageFingerprint: "stripeFingerprint",
		TokenizationMethod: "token",
		CustomerID:         customerID,
		StorageID:          "stripeCardStorageID",
		Type:               dal.PaymentMethodTypeCard,
	}, nil))

	// Move payment on to the next phase
	dmock.Expect(mock.NewExpectation(dmock.UpdatePayment, paymentID, &dal.PaymentUpdate{
		Lifecycle:   dal.PaymentLifecycleProcessing,
		ChangeState: dal.PaymentChangeStatePending,
	}))

	w := New(dmock, directorymock, nil, smock, "", "", "")
	w.processPaymentNoneAccepted()
}

func TestPaymentNoneAccepted_CustomerAlreadyExists(t *testing.T) {
	dmock := testutil.NewMockDAL(t)
	defer dmock.Finish()
	smock := testutil.NewMockIdempotentStripeClient(t)
	defer smock.Finish()
	directorymock := dirmock.New(t)
	defer directorymock.Finish()

	paymentID, err := dal.NewPaymentID()
	test.OK(t, err)
	vendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	masterVendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	masterPaymentMethodID, err := dal.NewPaymentMethodID()
	test.OK(t, err)
	masterCustomerID, err := dal.NewCustomerID()
	test.OK(t, err)
	customerID, err := dal.NewCustomerID()
	test.OK(t, err)

	// look for payments to processs
	dmock.Expect(mock.NewExpectation(
		dmock.PaymentsInState,
		dal.PaymentLifecycleAccepted,
		dal.PaymentChangeStateNone,
		int64(10),
		[]dal.QueryOption{dal.ForUpdate}).WithReturns([]*dal.Payment{
		{
			ID:              paymentID,
			VendorAccountID: vendorAccountID,
			PaymentMethodID: masterPaymentMethodID,
		},
	}, nil))

	// lookup vendored account that the payment method method is linked to
	dmock.Expect(mock.NewExpectation(dmock.VendorAccount, vendorAccountID).WithReturns(&dal.VendorAccount{
		ID:                 vendorAccountID,
		AccountType:        dal.VendorAccountAccountTypeStripe,
		ConnectedAccountID: "stripeConnectedAccountID",
	}, nil))

	// lookup the payment method (which is currently linked to the masterVendorAccount rather than the customer vendor account)
	dmock.Expect(mock.NewExpectation(dmock.PaymentMethod, masterPaymentMethodID, []dal.QueryOption(nil)).WithReturns(&dal.PaymentMethod{
		ID:                 masterPaymentMethodID,
		EntityID:           "entityID",
		VendorAccountID:    masterVendorAccountID,
		StorageFingerprint: "stripeFingerprint",
		TokenizationMethod: "token",
		CustomerID:         masterCustomerID,
		StorageID:          "masterStripeCardStorageID",
		Type:               dal.PaymentMethodTypeCard,
	}, nil))

	// look up an existing customer for the vendor which should come back as non-existent
	dmock.Expect(mock.NewExpectation(dmock.CustomerForVendor, vendorAccountID, "entityID", []dal.QueryOption(nil)).WithReturns(&dal.Customer{
		ID:              customerID,
		StorageType:     dal.CustomerStorageTypeStripe,
		StorageID:       "stripeCustomerID",
		VendorAccountID: vendorAccountID,
		EntityID:        "entityID",
		Lifecycle:       dal.CustomerLifecycleActive,
		ChangeState:     dal.CustomerChangeStateNone,
	}, nil))

	// lookup the card based on the fingerprint for the customer associated with the vendor
	dmock.Expect(mock.NewExpectation(
		dmock.PaymentMethodWithFingerprint,
		customerID,
		"stripeFingerprint", "token",
		[]dal.QueryOption(nil)).WithReturns((*dal.PaymentMethod)(nil), dal.ErrNotFound))

	// lookup the master customer
	dmock.Expect(mock.NewExpectation(dmock.Customer, masterCustomerID, []dal.QueryOption(nil)).WithReturns(&dal.Customer{
		StorageID: "masterCustomerStripeID",
	}, nil))

	// lookup again to ensure that card is not added
	dmock.Expect(mock.NewExpectation(
		dmock.PaymentMethodWithFingerprint,
		customerID,
		"newCardFingerprint", "newCardToken",
		[]dal.QueryOption(nil)).WithReturns((*dal.PaymentMethod)(nil), dal.ErrNotFound))

	paymentMethodID, err := dal.NewPaymentMethodID()
	test.OK(t, err)

	// create new card for customer
	dmock.Expect(mock.NewExpectation(dmock.InsertPaymentMethod, &dal.PaymentMethod{
		Type:               dal.PaymentMethodTypeCard,
		StorageType:        dal.PaymentMethodStorageTypeStripe,
		StorageID:          "stripeCardID",
		StorageFingerprint: "newCardFingerprint",
		Brand:              "Visa",
		Last4:              "1234",
		ExpMonth:           int(10),
		ExpYear:            int(24),
		TokenizationMethod: "newCardToken",
		VendorAccountID:    vendorAccountID,
		CustomerID:         customerID,
		EntityID:           "entityID",
		Lifecycle:          dal.PaymentMethodLifecycleActive,
		ChangeState:        dal.PaymentMethodChangeStateNone,
	}).WithReturns(paymentMethodID, nil))

	// map the existing payment method to the newly created payment method
	dmock.Expect(mock.NewExpectation(dmock.UpdatePaymentMethod, paymentID, &dal.PaymentUpdate{
		Lifecycle:       dal.PaymentLifecycleAccepted,
		ChangeState:     dal.PaymentChangeStateNone,
		PaymentMethodID: &paymentMethodID,
	}))

	// Move payment on to the next phase
	dmock.Expect(mock.NewExpectation(dmock.UpdatePayment, paymentID, &dal.PaymentUpdate{
		Lifecycle:   dal.PaymentLifecycleProcessing,
		ChangeState: dal.PaymentChangeStatePending,
	}))

	smock.Expect(mock.NewExpectation(smock.Token, &stripe.TokenParams{
		Customer: "masterCustomerStripeID",
		Card:     &stripe.CardParams{Token: "masterStripeCardStorageID"},
		Params: stripe.Params{
			StripeAccount: "stripeConnectedAccountID",
		},
	}).WithReturns(&stripe.Token{
		ID: "token",
	}, nil))

	smock.Expect(mock.NewExpectation(smock.CreateCard, &stripe.CardParams{
		Customer: "stripeCustomerID",
		Token:    "token",
		Params: stripe.Params{
			StripeAccount: "stripeConnectedAccountID",
			Meta: map[string]string{
				"entity_id":   "entityID",
				"customer_id": customerID.String(),
			},
		},
	}).WithReturns(&stripe.Card{
		ID:                 "stripeCardID",
		Fingerprint:        "newCardFingerprint",
		TokenizationMethod: stripe.TokenizationMethod("newCardToken"),
		Brand:              stripe.CardBrand("Visa"),
		Month:              uint8(10),
		Year:               uint16(24),
		LastFour:           "1234",
	}, nil))

	w := New(dmock, directorymock, nil, smock, "", "", "")
	w.processPaymentNoneAccepted()
}

func TestPaymentNoneAccepted_CustomerAndPaymentMethodAlreadyExists(t *testing.T) {
	dmock := testutil.NewMockDAL(t)
	defer dmock.Finish()
	smock := testutil.NewMockIdempotentStripeClient(t)
	defer smock.Finish()
	directorymock := dirmock.New(t)
	defer directorymock.Finish()

	paymentID, err := dal.NewPaymentID()
	test.OK(t, err)
	vendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	masterVendorAccountID, err := dal.NewVendorAccountID()
	test.OK(t, err)
	masterPaymentMethodID, err := dal.NewPaymentMethodID()
	test.OK(t, err)
	masterCustomerID, err := dal.NewCustomerID()
	test.OK(t, err)
	customerID, err := dal.NewCustomerID()
	test.OK(t, err)
	paymentMethodID, err := dal.NewPaymentMethodID()
	test.OK(t, err)

	// look for payments to processs
	dmock.Expect(mock.NewExpectation(
		dmock.PaymentsInState,
		dal.PaymentLifecycleAccepted,
		dal.PaymentChangeStateNone,
		int64(10),
		[]dal.QueryOption{dal.ForUpdate}).WithReturns([]*dal.Payment{
		{
			ID:              paymentID,
			VendorAccountID: vendorAccountID,
			PaymentMethodID: masterPaymentMethodID,
		},
	}, nil))

	// lookup vendored account that the payment method method is linked to
	dmock.Expect(mock.NewExpectation(dmock.VendorAccount, vendorAccountID).WithReturns(&dal.VendorAccount{
		ID:                 vendorAccountID,
		AccountType:        dal.VendorAccountAccountTypeStripe,
		ConnectedAccountID: "stripeConnectedAccountID",
	}, nil))

	// lookup the payment method (which is currently linked to the masterVendorAccount rather than the customer vendor account)
	dmock.Expect(mock.NewExpectation(dmock.PaymentMethod, masterPaymentMethodID, []dal.QueryOption(nil)).WithReturns(&dal.PaymentMethod{
		ID:                 masterPaymentMethodID,
		EntityID:           "entityID",
		VendorAccountID:    masterVendorAccountID,
		StorageFingerprint: "stripeFingerprint",
		TokenizationMethod: "token",
		CustomerID:         masterCustomerID,
		StorageID:          "masterStripeCardStorageID",
		Type:               dal.PaymentMethodTypeCard,
	}, nil))

	// look up an existing customer for the vendor which should come back as non-existent
	dmock.Expect(mock.NewExpectation(dmock.CustomerForVendor, vendorAccountID, "entityID", []dal.QueryOption(nil)).WithReturns(&dal.Customer{
		ID:              customerID,
		StorageType:     dal.CustomerStorageTypeStripe,
		StorageID:       "stripeCustomerID",
		VendorAccountID: vendorAccountID,
		EntityID:        "entityID",
		Lifecycle:       dal.CustomerLifecycleActive,
		ChangeState:     dal.CustomerChangeStateNone,
	}, nil))

	// lookup the card based on the fingerprint for the customer associated with the vendor
	dmock.Expect(mock.NewExpectation(
		dmock.PaymentMethodWithFingerprint,
		customerID,
		"stripeFingerprint", "token",
		[]dal.QueryOption(nil)).WithReturns(&dal.PaymentMethod{
		ID:                 paymentMethodID,
		Type:               dal.PaymentMethodTypeCard,
		StorageType:        dal.PaymentMethodStorageTypeStripe,
		StorageID:          "stripeCardID",
		StorageFingerprint: "newCardFingerprint",
		Brand:              "Visa",
		Last4:              "1234",
		ExpMonth:           int(10),
		ExpYear:            int(24),
		TokenizationMethod: "newCardToken",
		VendorAccountID:    vendorAccountID,
		CustomerID:         customerID,
		EntityID:           "entityID",
		Lifecycle:          dal.PaymentMethodLifecycleActive,
		ChangeState:        dal.PaymentMethodChangeStateNone,
	}, nil))

	// Move payment on to the next phase
	dmock.Expect(mock.NewExpectation(dmock.UpdatePayment, paymentID, &dal.PaymentUpdate{
		Lifecycle:   dal.PaymentLifecycleProcessing,
		ChangeState: dal.PaymentChangeStatePending,
	}))

	w := New(dmock, directorymock, nil, smock, "", "", "")
	w.processPaymentNoneAccepted()
}

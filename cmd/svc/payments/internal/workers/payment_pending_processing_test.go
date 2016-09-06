package workers

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/testutil"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/threading"
	threadmock "github.com/sprucehealth/backend/svc/threading/mock"
	"github.com/stripe/stripe-go"
)

func TestPaymentPendingProcessing(t *testing.T) {
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
		dal.PaymentLifecycleProcessing,
		dal.PaymentChangeStatePending,
		int64(10),
		[]dal.QueryOption{dal.ForUpdate}).WithReturns([]*dal.Payment{
		{
			ID:              paymentID,
			VendorAccountID: vendorAccountID,
			PaymentMethodID: paymentMethodID,
			Amount:          1000,
			Currency:        "USD",
		},
	}, nil))

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

	dmock.Expect(mock.NewExpectation(dmock.Customer, customerID, []dal.QueryOption(nil)).WithReturns(&dal.Customer{
		StorageID: "customerStripeID",
	}, nil))

	sourceParams, err := stripe.SourceParamsFor("stripeCardStorageID")
	test.OK(t, err)

	smock.Expect(mock.NewExpectation(smock.CreateCharge, &stripe.ChargeParams{
		Amount:   1000,
		Currency: "USD",
		Source:   sourceParams,
		Customer: "customerStripeID",
		Params: stripe.Params{
			StripeAccount: "stripeConnectedAccountID",
		},
	}).WithReturns(&stripe.Charge{ID: "stripeChargeID"}, nil))

	dmock.Expect(mock.NewExpectation(dmock.UpdatePayment, paymentID, &dal.PaymentUpdate{
		Lifecycle:              dal.PaymentLifecycleCompleted,
		ChangeState:            dal.PaymentChangeStateNone,
		ProcessorTransactionID: ptr.String("stripeChargeID"),
	}))

	w := New(dmock, directorymock, nil, smock, "", "", "")
	w.processPaymentPendingProcessing()

}

func TestPaymentPendingProcessing_CardDeclined(t *testing.T) {
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
		dal.PaymentLifecycleProcessing,
		dal.PaymentChangeStatePending,
		int64(10),
		[]dal.QueryOption{dal.ForUpdate}).WithReturns([]*dal.Payment{
		{
			ID:              paymentID,
			VendorAccountID: vendorAccountID,
			PaymentMethodID: paymentMethodID,
			Amount:          1000,
			Currency:        "USD",
			ThreadID:        "threadID",
		},
	}, nil))

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

	dmock.Expect(mock.NewExpectation(dmock.Customer, customerID, []dal.QueryOption(nil)).WithReturns(&dal.Customer{
		StorageID: "customerStripeID",
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

	sourceParams, err := stripe.SourceParamsFor("stripeCardStorageID")
	test.OK(t, err)

	smock.Expect(mock.NewExpectation(smock.CreateCharge, &stripe.ChargeParams{
		Amount:   1000,
		Currency: "USD",
		Source:   sourceParams,
		Customer: "customerStripeID",
		Params: stripe.Params{
			StripeAccount: "stripeConnectedAccountID",
		},
	}).WithReturns((*stripe.Charge)(nil), &stripe.Error{
		Code: "card_declined",
		Type: stripe.ErrorTypeCard,
		Msg:  "Card was declined",
	}))

	dmock.Expect(mock.NewExpectation(dmock.UpdatePayment, paymentID, &dal.PaymentUpdate{
		Lifecycle:              dal.PaymentLifecycleErrorProcessing,
		ChangeState:            dal.PaymentChangeStateNone,
		ProcessorStatusMessage: ptr.String("Card was declined"),
	}))

	tmock := threadmock.New(t)
	defer tmock.Finish()

	var title bml.BML
	title = append(title, "Error Processing Payment: ")
	title = append(title, &bml.Anchor{
		HREF: deeplink.PaymentURL("test.com", "orgID", "threadID", paymentID.String()),
		Text: "Card was declined",
	})
	titleText, err := title.Format()
	test.OK(t, err)
	summary, err := title.PlainText()
	test.OK(t, err)

	tmock.Expect(mock.NewExpectation(tmock.Thread, &threading.ThreadRequest{
		ThreadID: "threadID",
	}).WithReturns(&threading.ThreadResponse{
		Thread: &threading.Thread{
			ID:             "threadID",
			OrganizationID: "orgID",
		},
	}, nil))

	tmock.Expect(mock.NewExpectation(tmock.PostMessage, &threading.PostMessageRequest{
		UUID:         `error_processing_` + paymentID.String(),
		ThreadID:     "threadID",
		FromEntityID: "entityID",
		Title:        titleText,
		Summary:      summary,
	}))

	w := New(dmock, directorymock, tmock, smock, "", "", "test.com")
	w.processPaymentPendingProcessing()

}

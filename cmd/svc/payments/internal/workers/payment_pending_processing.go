package workers

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/stripe/stripe-go"
)

// processPaymentPendingProcessing performs the actual processing of payments
func (w *Workers) processPaymentPendingProcessing() {
	ctx := context.Background()
	if err := w.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		paymentsPendingProcessing, err := dl.PaymentsInState(ctx, dal.PaymentLifecycleProcessing, dal.PaymentChangeStatePending, 10, dal.ForUpdate)
		if err != nil {
			return errors.Trace(err)
		}
		if len(paymentsPendingProcessing) == 0 {
			return nil
		}
		golog.Debugf("Found %d payments PENDING/PROCESSING", len(paymentsPendingProcessing))
		for _, p := range paymentsPendingProcessing {
			vendorAccount, err := dl.VendorAccount(ctx, p.VendorAccountID)
			if err != nil {
				golog.Errorf("Error while looking up vendor account %s for payment %s", p.VendorAccountID, p.ID)
				continue
			}
			paymentMethod, err := dl.PaymentMethod(ctx, p.PaymentMethodID)
			if err != nil {
				golog.Errorf("Error while looking up payment method %s for payment %s", p.PaymentMethodID, p.ID)
				continue
			}
			customer, err := dl.Customer(ctx, paymentMethod.CustomerID)
			if err != nil {
				golog.Errorf("Error while looking up customer %s for payment method %s", paymentMethod.CustomerID, paymentMethod.ID)
				continue
			}
			switch vendorAccount.AccountType {
			case dal.VendorAccountAccountTypeStripe:
				sourceParams, err := stripe.SourceParamsFor(paymentMethod.StorageID)
				if err != nil {
					golog.Errorf("Error while creating Stripe source params for payment method %s: %s", paymentMethod.ID, err)
					continue
				}
				charge, err := w.stripeClient.CreateCharge(ctx, &stripe.ChargeParams{
					Amount:   p.Amount,
					Currency: stripe.Currency(p.Currency),
					Source:   sourceParams,
					Customer: customer.StorageID,
					Params: stripe.Params{
						StripeAccount: vendorAccount.ConnectedAccountID,
					},
				})
				if err != nil {
					golog.Errorf("Error while creating Stripe charge for payment %s: %s", p.ID, err)
					continue
				}
				golog.Debugf("Created charge %+v", charge)
			default:
				golog.Errorf("Unsupported vendor account type %s for vendor account %s and payment %s", vendorAccount.AccountType, p.VendorAccountID, p.ID)
				continue
			}
			if _, err := dl.UpdatePayment(ctx, p.ID, &dal.PaymentUpdate{
				Lifecycle:   dal.PaymentLifecycleCompleted,
				ChangeState: dal.PaymentChangeStateNone,
			}); err != nil {
				golog.Errorf("Error while updating payment %s with new paymentMethodID: %s", p.ID, err)
				continue
			}
			golog.Debugf("Payment %s processed", p.ID)
		}
		return nil
	}); err != nil {
		golog.Errorf("Encountered error while processing PENDING/PROCESSING payments: %s", err)
	}
}

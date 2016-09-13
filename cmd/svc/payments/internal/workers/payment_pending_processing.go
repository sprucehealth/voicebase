package workers

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/server"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/smet"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/threading"
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
				smet.Errorf(workerErrMetricName, "Error while looking up vendor account %s for payment %s", p.VendorAccountID, p.ID)
				continue
			}
			paymentMethod, err := dl.PaymentMethod(ctx, p.PaymentMethodID)
			if err != nil {
				smet.Errorf(workerErrMetricName, "Error while looking up payment method %s for payment %s", p.PaymentMethodID, p.ID)
				continue
			}
			customer, err := dl.Customer(ctx, paymentMethod.CustomerID)
			if err != nil {
				smet.Errorf(workerErrMetricName, "Error while looking up customer %s for payment method %s", paymentMethod.CustomerID, paymentMethod.ID)
				continue
			}
			var processingErr error
			var processorTransactionID *string
			switch vendorAccount.AccountType {
			case dal.VendorAccountAccountTypeStripe:
				var sourceParams *stripe.SourceParams
				sourceParams, processingErr = stripe.SourceParamsFor(paymentMethod.StorageID)
				if processingErr != nil {
					processingErr = fmt.Errorf("Error while creating Stripe source params for payment method %s: %s", paymentMethod.ID, err)
					break
				}
				var charge *stripe.Charge
				charge, processingErr = w.stripeClient.CreateCharge(ctx, &stripe.ChargeParams{
					Amount:   p.Amount,
					Currency: stripe.Currency(p.Currency),
					Source:   sourceParams,
					Customer: customer.StorageID,
					Params: stripe.Params{
						StripeAccount: vendorAccount.ConnectedAccountID,
					},
				})
				if processingErr != nil {
					break
				}
				processorTransactionID = &charge.ID
				golog.Debugf("Created stripe charge %+v", charge)
			default:
				smet.Errorf(workerErrMetricName, "Unsupported vendor account type %s for vendor account %s and payment %s", vendorAccount.AccountType, p.VendorAccountID, p.ID)
				continue
			}
			if processingErr != nil {
				if server.IsPaymentMethodError(errors.Cause(processingErr)) {
					if server.IsPaymentMethodErrorRetryable(errors.Cause(processingErr)) {
						golog.Infof("Encountered retryable error while processing payment %s: %s", p.ID, processingErr)
						continue
					} else {
						processingErrReason := server.PaymentMethodErrorMesssage(errors.Cause(processingErr))
						if err := w.postErrorProcessingToThread(ctx, p, processingErrReason); err != nil {
							smet.Errorf(workerErrMetricName, "Error while posting processing error thread update to thread %s for payment %s: %s", p.ThreadID, p.ID, err)
							continue
						}
						if _, err := dl.UpdatePayment(ctx, p.ID, &dal.PaymentUpdate{
							Lifecycle:              dal.PaymentLifecycleErrorProcessing,
							ChangeState:            dal.PaymentChangeStateNone,
							ProcessorStatusMessage: ptr.String(processingErrReason),
						}); err != nil {
							smet.Errorf(workerErrMetricName, "Error while updating payment %s with new processorStatusMessage %q: %s", p.ID, server.PaymentMethodErrorMesssage(errors.Cause(processingErr)), err)
						}
						continue
					}
				}
				smet.Errorf(workerErrMetricName, "Error while processing payment %s: %s", p.ID, processingErr)
				continue
			} else {

				if err := w.postPaymentCompletedToThread(ctx, p); err != nil {
					smet.Errorf(workerErrMetricName, "Unable to post completed message to thread for payment %s : %s", p.ID, err)
					continue
				}

				if _, err := dl.UpdatePayment(ctx, p.ID, &dal.PaymentUpdate{
					Lifecycle:              dal.PaymentLifecycleCompleted,
					ChangeState:            dal.PaymentChangeStateNone,
					ProcessorTransactionID: processorTransactionID,
				}); err != nil {
					smet.Errorf(workerErrMetricName, "Error while updating payment %s with new processorTransactionID %s: %s", p.ID, *processorTransactionID, err)
					continue
				}
				golog.Debugf("Payment %s processed", p.ID)
			}
		}
		return nil
	}); err != nil {
		smet.Errorf(workerErrMetricName, "Encountered error while processing PENDING/PROCESSING payments: %s", err)
	}
}

func (w *Workers) postPaymentCompletedToThread(ctx context.Context, payment *dal.Payment) error {
	if payment.ThreadID == "" {
		return nil
	}

	resp, err := w.threadingClient.Thread(ctx, &threading.ThreadRequest{
		ThreadID: payment.ThreadID,
	})
	if err != nil {
		return errors.Trace(err)
	}

	paymentMethod, err := w.dal.PaymentMethod(ctx, payment.PaymentMethodID)
	if err != nil {
		return errors.Trace(err)
	}

	var title bml.BML
	title = append(title, "Completed Payment: ")
	title = append(title, &bml.Anchor{
		HREF: deeplink.PaymentURL(w.webDomain, resp.Thread.OrganizationID, resp.Thread.ID, payment.ID.String()),
		Text: payments.FormatAmount(payment.Amount, "USD"),
	})
	titleText, err := title.Format()
	if err != nil {
		return errors.Trace(err)
	}
	summary, err := title.PlainText()
	if err != nil {
		return errors.Trace(err)
	}
	if _, err = w.threadingClient.PostMessage(ctx, &threading.PostMessageRequest{
		UUID:         `accept_` + payment.ID.String(),
		ThreadID:     payment.ThreadID,
		FromEntityID: paymentMethod.EntityID,
		Title:        titleText,
		Summary:      summary,
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (w *Workers) postErrorProcessingToThread(ctx context.Context, payment *dal.Payment, errorMessage string) error {
	if payment.ThreadID == "" {
		return nil
	}
	paymentMethod, err := w.dal.PaymentMethod(ctx, payment.PaymentMethodID)
	if err != nil {
		return errors.Trace(err)
	}
	resp, err := w.threadingClient.Thread(ctx, &threading.ThreadRequest{
		ThreadID: payment.ThreadID,
	})
	if err != nil {
		return errors.Trace(err)
	}
	var title bml.BML
	title = append(title, "Error Processing Payment: ")
	title = append(title, &bml.Anchor{
		HREF: deeplink.PaymentURL(w.webDomain, resp.Thread.OrganizationID, resp.Thread.ID, payment.ID.String()),
		Text: errorMessage,
	})
	titleText, err := title.Format()
	if err != nil {
		return errors.Trace(err)
	}
	summary, err := title.PlainText()
	if err != nil {
		return errors.Trace(err)
	}
	if _, err = w.threadingClient.PostMessage(ctx, &threading.PostMessageRequest{
		UUID:     `error_processing_` + payment.ID.String(),
		ThreadID: payment.ThreadID,
		// TODO: For now just assume whoever owns the payment method accepted it
		FromEntityID: paymentMethod.EntityID,
		Title:        titleText,
		Summary:      summary,
	}); err != nil {
		return errors.Trace(err)
	}
	return nil
}

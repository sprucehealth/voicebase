package workers

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/server"
	istripe "github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/stripe/stripe-go"
)

// processPaymentNoneAccepted asserts the existance of the customer and payment method in the context of the vendor
func (w *Workers) processPaymentNoneAccepted() {
	ctx := context.Background()
	if err := w.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		paymentsNoneAccepted, err := dl.PaymentsInState(ctx, dal.PaymentLifecycleAccepted, dal.PaymentChangeStateNone, 10, dal.ForUpdate)
		if err != nil {
			return errors.Trace(err)
		}
		if len(paymentsNoneAccepted) == 0 {
			return nil
		}
		golog.Debugf("Found %d payments NONE/ACCEPTED", len(paymentsNoneAccepted))
		for _, p := range paymentsNoneAccepted {
			golog.Debugf("Attempting customer/payment_method processing of payment: %s", p.ID)
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
			// If the payment method isn't mapped to this vendor then resolve it
			if paymentMethod.VendorAccountID != vendorAccount.ID {
				// Check to see if the customer exists for this vendor
				vendorCustomer, err := dl.CustomerForVendor(ctx, p.VendorAccountID, paymentMethod.EntityID)
				if errors.Cause(err) == dal.ErrNotFound {
					// If we can't find the customer for this vendor then add it.
					// TODO: Source the customer addition from a token rather than the entityID - https://app.asana.com/0/163568593435416/170802620739346
					vendorCustomer, err = server.AddCustomer(ctx, vendorAccount, paymentMethod.EntityID, w.dal, w.directoryClient, w.stripeClient)
					if err != nil {
						golog.Errorf("Error while adding customer for vendor account %s and entity id %s for payment %s: %s", p.VendorAccountID, paymentMethod.EntityID, p.ID, err)
						continue
					}

					// link the stripe customer to the entity
					if _, err := w.directoryClient.CreateExternalLink(ctx, &directory.CreateExternalLinkRequest{
						EntityID: paymentMethod.EntityID,
						Name:     "Stripe",
						URL:      fmt.Sprintf("%s/customers/%s", istripe.DashboardURL(), vendorCustomer.StorageID),
					}); err != nil {
						golog.Errorf("Unable to create external link for %s : %s", paymentMethod.EntityID, err)
						continue
					}

					if _, err := w.directoryClient.CreateExternalIDs(ctx, &directory.CreateExternalIDsRequest{
						EntityID:    paymentMethod.EntityID,
						ExternalIDs: []string{scopeID(vendorCustomer.StorageID)},
					}); err != nil {
						golog.Errorf("Unable to create externalIDs for %s : %s", paymentMethod.EntityID, err)
						continue
					}

				} else if err != nil {
					golog.Errorf("Error while looking up customer for vendor account %s and entity id %s for payment %s: %s", p.VendorAccountID, paymentMethod.EntityID, p.ID, err)
					continue
				}
				// Check to see if there is an existing matching payment method for this vendor customer
				vendorPaymentMethod, err := dl.PaymentMethodWithFingerprint(
					ctx,
					vendorCustomer.ID,
					paymentMethod.StorageFingerprint,
					paymentMethod.TokenizationMethod)
				if errors.Cause(err) == dal.ErrNotFound {
					// Create the token source for adding this card
					var tokenSource server.TokenSource
					switch vendorAccount.AccountType {
					case dal.VendorAccountAccountTypeStripe:
						customer, err := dl.Customer(ctx, paymentMethod.CustomerID)
						if err != nil {
							golog.Errorf("Error while looking up customer %s for payment method %s", paymentMethod.CustomerID, paymentMethod.ID)
							continue
						}
						tokenSource = &server.DynamicTokenSource{
							D: func() (string, error) {
								token, err := w.stripeClient.Token(ctx, &stripe.TokenParams{
									Customer: customer.StorageID,
									Card:     &stripe.CardParams{Token: paymentMethod.StorageID},
									Params: stripe.Params{
										StripeAccount: vendorAccount.ConnectedAccountID,
									},
								})
								if err != nil {
									return "", errors.Trace(err)
								}
								return token.ID, nil
							},
						}
					default:
						golog.Errorf("Unsupported vendor account type %s for vendor account %s and payment %s", vendorAccount.AccountType, p.VendorAccountID, p.ID)
						continue
					}
					//sanity
					if tokenSource == nil {
						golog.Errorf("Nil token source for payment %s - This should never happen", p.ID)
						continue
					}

					// TODO: Assert that we can make this idempotent with new tokens each attempt
					vendorPaymentMethod, err = server.AddPaymentMethod(
						ctx,
						vendorAccount,
						vendorCustomer,
						server.TransformPaymentMethodTypeToResponse(paymentMethod.Type),
						tokenSource,
						w.dal,
						w.stripeClient)
					if err != nil {
						golog.Errorf("Error while adding payment method for vendor account %s and entity id %s for payment %s: %s", p.VendorAccountID, paymentMethod.EntityID, p.ID, err)
						continue
					}

					// Map our payment to the newly tracked payment method
					if _, err := dl.UpdatePayment(ctx, p.ID, &dal.PaymentUpdate{
						Lifecycle:       dal.PaymentLifecycleAccepted,
						ChangeState:     dal.PaymentChangeStateNone,
						PaymentMethodID: &vendorPaymentMethod.ID,
					}); err != nil {
						golog.Errorf("Error while updating payment %s with new paymentMethodID: %s", p.ID, err)
						continue
					}
				} else if err != nil {
					golog.Errorf("Error while looking up customer for vendor account %s and entity id %s for payment %s: %s", p.VendorAccountID, paymentMethod.EntityID, p.ID, err)
					continue
				}
			} else {
				golog.Debugf("Payment PaymentMethod is already mapped to vendor, no resolve required")
			}
			// Move our payment on to the next phase
			if _, err := dl.UpdatePayment(ctx, p.ID, &dal.PaymentUpdate{
				Lifecycle:   dal.PaymentLifecycleProcessing,
				ChangeState: dal.PaymentChangeStatePending,
			}); err != nil {
				golog.Errorf("Error while updating payment %s with new paymentMethodID: %s", p.ID, err)
				continue
			}
			golog.Debugf("Payment %s customer/payment_method processed", p.ID)
		}
		return nil
	}); err != nil {
		golog.Errorf("Encountered error while processing NONE/ACCEPTED payments: %s", err)
	}
}

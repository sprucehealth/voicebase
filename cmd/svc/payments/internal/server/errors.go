package server

import (
	istripe "github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/stripe/stripe-go"
)

// IsPaymentMethodError returns a boolean value representing if a payment processor claims the payment method is in error
func IsPaymentMethodError(err error) bool {
	if isStripeCardErr(errors.Cause(err)) {
		return true
	}
	return false
}

func isStripeCardErr(err error) bool {
	return istripe.ErrType(err) == stripe.ErrorTypeCard
}

// PaymentMethodErrorMesssage returns the well formtted message inside a payment method error
func PaymentMethodErrorMesssage(err error) string {
	if isStripeCardErr(errors.Cause(err)) {
		return istripe.ErrMessage(err)
	}
	return ""
}

// IsProcessingErrorRetryable returns is the error related to payment processing is retryable
func IsProcessingErrorRetryable(err error) bool {
	return isStripeErrorRetryable(errors.Cause(err))
}

func isStripeErrorRetryable(err error) bool {
	return istripe.ErrCode(err) == stripe.ProcessingErr ||
		// TODO: Backoff
		istripe.ErrCode(err) == stripe.RateLimit
}

package server

import (
	istripe "github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/stripe/stripe-go"
)

// IsPaymentMethodError returns a boolean value representing if a payment processor claims the payment method is in error
func IsPaymentMethodError(err error) bool {
	if isStripeCardErr(err) {
		return true
	}
	return false
}

func isStripeCardErr(err error) bool {
	return istripe.ErrType(err) == stripe.ErrorTypeCard
}

// PaymentMethodErrorMesssage returns the well formtted message inside a payment method error
func PaymentMethodErrorMesssage(err error) string {
	if isStripeCardErr(err) {
		return istripe.ErrMessage(err)
	}
	return ""
}

// IsPaymentMethodErrorRetryable returns is the error related to the payment method is retryable
func IsPaymentMethodErrorRetryable(err error) bool {
	if isStripeCardErr(err) {
		return isStripeCardErrorRetryable(err)
	}
	return false
}

func isStripeCardErrorRetryable(err error) bool {
	return istripe.ErrCode(err) == stripe.ProcessingErr
}

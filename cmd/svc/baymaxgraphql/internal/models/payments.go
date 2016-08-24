package models

import "github.com/sprucehealth/backend/svc/payments"

// PaymentMethod represents the common payment method interface
type PaymentMethod interface {
	PaymentMethodType() payments.PaymentMethodType
}

// PaymentCard represents a card payment method
type PaymentCard struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Default            bool   `json:"default"`
	PaymentProcessor   string `json:"paymentProcessor"`
	TokenizationMethod string `json:"tokenizationMethod"`
	Brand              string `json:"brand"`
	Last4              string `json:"last4"`
}

// PaymentMethodType returns the type of the payment method
func (p *PaymentCard) PaymentMethodType() payments.PaymentMethodType {
	return payments.PAYMENT_METHOD_TYPE_CARD
}

// PaymentRequest represents a card payment method
type PaymentRequest struct {
	ID                 string        `json:"id"`
	RequestingEntity   *Entity       `json:"requestingEntity"`
	PaymentMethod      PaymentMethod `json:"paymentMethod"`
	AmountInCents      uint64        `json:"amountInCents"`
	Status             string        `json:"status"`
	Currency           string        `json:"currency"`
	RequestedTimestamp uint64        `json:"requestedTimestamp"`
	CompletedTimestamp *uint64       `json:"completedTimestamp"`
}

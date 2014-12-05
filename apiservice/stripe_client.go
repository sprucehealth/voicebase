package apiservice

import "github.com/sprucehealth/backend/libs/stripe"

// StripeClient is an interface wrapper for the actual stripe client
// which any handler in the restapi uses. This makes it easy to stub out the
// actual stripe client thereby making it possible to run integration tests
// without requiring to talk to stripe
type StripeClient interface {
	AddCardForCustomer(cardToken, customerId string) (*stripe.Card, error)
	CreateChargeForCustomer(req *stripe.CreateChargeRequest) (*stripe.Charge, error)
	CreateCustomerWithDefaultCard(token string) (*stripe.Customer, error)
	DeleteCardForCustomer(customerId string, cardId string) error
	GetCardsForCustomer(customerId string) ([]*stripe.Card, error)
	ListAllCharges(limit int) ([]*stripe.Charge, error)
	ListAllCustomerCharges(customerID string) ([]*stripe.Charge, error)
	MakeCardDefaultForCustomer(cardId, customerId string) error
}

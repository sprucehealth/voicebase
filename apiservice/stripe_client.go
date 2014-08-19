package apiservice

import "github.com/sprucehealth/backend/libs/stripe"

// StripeClient is an interface wrapper for the actual stripe client
// which any handler in the restapi uses. This makes it easy to stub out the
// actual stripe client thereby making it possible to run integration tests
// without requiring to talk to stripe
type StripeClient interface {
	CreateCustomerWithDefaultCard(token string) (*stripe.Customer, error)
	AddCardForCustomer(cardToken, customerId string) (*stripe.Card, error)
	MakeCardDefaultForCustomer(cardId, customerId string) error
	GetCardsForCustomer(customerId string) ([]*stripe.Card, error)
	DeleteCardForCustomer(customerId string, cardId string) error
	CreateChargeForCustomer(req *stripe.CreateChargeRequest) (*stripe.Charge, error)
	ListAllCustomerCharges(customerID string) ([]*stripe.Charge, error)
}

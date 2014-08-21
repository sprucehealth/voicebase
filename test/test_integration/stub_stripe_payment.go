package test_integration

import "github.com/sprucehealth/backend/libs/stripe"

type StripeStub struct {
	CustomerToReturn   *stripe.Customer
	CardToReturnOnAdd  *stripe.Card
	CardsToReturn      []*stripe.Card
	CreateChargeFunc   func(req *stripe.CreateChargeRequest) (*stripe.Charge, error)
	ListAllChargesFunc func(string) ([]*stripe.Charge, error)
}

func (s *StripeStub) CreateCustomerWithDefaultCard(token string) (*stripe.Customer, error) {
	return s.CustomerToReturn, nil
}

func (s *StripeStub) AddCardForCustomer(cardToken string, customerId string) (*stripe.Card, error) {
	return s.CardToReturnOnAdd, nil
}

func (s *StripeStub) MakeCardDefaultForCustomer(cardId string, customerId string) error {
	return nil
}

func (s *StripeStub) GetCardsForCustomer(customerId string) ([]*stripe.Card, error) {
	return s.CardsToReturn, nil
}

func (s *StripeStub) DeleteCardForCustomer(customerId string, cardId string) error {
	return nil
}

func (s *StripeStub) CreateChargeForCustomer(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
	return s.CreateChargeFunc(req)
}

func (s *StripeStub) ListAllCustomerCharges(customerID string) ([]*stripe.Charge, error) {
	if s.ListAllChargesFunc != nil {
		return s.ListAllChargesFunc(customerID)
	}
	return nil, nil
}

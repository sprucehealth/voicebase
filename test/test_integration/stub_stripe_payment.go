package test_integration

import "github.com/sprucehealth/backend/libs/stripe"

type StripeStub struct {
	CustomerToReturn  *stripe.Customer
	CardToReturnOnAdd *stripe.Card
	CardsToReturn     []*stripe.Card
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
	return nil, nil
}

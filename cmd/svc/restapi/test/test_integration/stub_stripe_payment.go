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

func (s *StripeStub) AddCardForCustomer(cardToken string, customerID string) (*stripe.Card, error) {
	return s.CardToReturnOnAdd, nil
}

func (s *StripeStub) MakeCardDefaultForCustomer(cardID string, customerID string) error {
	return nil
}

func (s *StripeStub) GetCardsForCustomer(customerID string) ([]*stripe.Card, error) {
	return s.CardsToReturn, nil
}

func (s *StripeStub) DeleteCardForCustomer(customerID string, cardID string) error {
	return nil
}

func (s *StripeStub) CreateChargeForCustomer(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
	if s.CreateChargeFunc != nil {
		return s.CreateChargeFunc(req)
	}

	return nil, nil
}

func (s *StripeStub) ListAllCharges(limit int) ([]*stripe.Charge, error) {
	return nil, nil
}

func (s *StripeStub) ListAllCustomerCharges(customerID string) ([]*stripe.Charge, error) {
	if s.ListAllChargesFunc != nil {
		return s.ListAllChargesFunc(customerID)
	}
	return nil, nil
}

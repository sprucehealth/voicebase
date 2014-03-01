package payment

import "carefront/common"

type PaymentAPI interface {
	CreateCustomerWithDefaultCard(token string) (*Customer, error)
	AddCardForCustomer(cardToken, customerId string) (*common.Card, error)
	MakeCardDefaultForCustomer(cardId, customerId string) error
	GetCardsForCustomer(customerId string) ([]common.Card, error)
	DeleteCardForCustomer(customerId string, cardId string) error
}

type Customer struct {
	Id    string
	Cards []common.Card
}

package stripe

import (
	"encoding/json"
	"fmt"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/payment"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	stripeUrl          = "https://api.stripe.com/v1/"
	stripeCustomersUrl = stripeUrl + "customers"
	apiVersion         = "2014-01-31"
)

type StripeService struct {
	SecretKey string
}

type stripeCustomer struct {
	Id    string         `json:"id"`
	Cards stripeCardData `json:"cards"`
}

type stripeCard struct {
	Id          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Type        string `json:"type"`
	ExpMonth    int64  `json:"exp_month"`
	ExpYear     int64  `json:"exp_year"`
	Last4       int64  `json:"last4,string"`
}

type stripeCardData struct {
	Count int64         `json:"count"`
	Data  []*stripeCard `json:"data"`
}

type StripeError struct {
	Code    int
	Details struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (s StripeError) Error() string {
	return fmt.Sprintf("Error communicating with stripe. ErrorCode: %dErrorDetails:\n- Type: %s\n- Message: %s\n- Code:%s\n", s.Code, s.Details.Type, s.Details.Message, s.Details.Code)
}

func (s *StripeService) CreateCustomerWithDefaultCard(token string) (*payment.Customer, error) {
	params := url.Values{}
	params.Set("card", token)

	sCustomer := &stripeCustomer{}
	if err := s.query("POST", stripeCustomersUrl, params, sCustomer); err != nil {
		return nil, err
	}

	if sCustomer.Cards.Count == 0 {
		return nil, fmt.Errorf("Expected atleast 1 card to be returned when creating the customer")
	}

	return &payment.Customer{
		Id: sCustomer.Id,
		Cards: []common.Card{common.Card{
			ThirdPartyId: sCustomer.Cards.Data[0].Id,
			Fingerprint:  sCustomer.Cards.Data[0].Fingerprint,
		},
		},
	}, nil
}

func (s *StripeService) GetCardsForCustomer(customerId string) ([]*common.Card, error) {
	sCardData := &stripeCardData{}
	if err := s.query("GET", fmt.Sprintf("%s/%s/cards", stripeCustomersUrl, customerId), nil, sCardData); err != nil {
		return nil, err
	}

	cards := make([]*common.Card, len(sCardData.Data))
	for i, card := range sCardData.Data {
		cards[i] = &common.Card{
			ThirdPartyId: card.Id,
			ExpMonth:     card.ExpMonth,
			ExpYear:      card.ExpYear,
			Last4:        card.Last4,
			Type:         card.Type,
			Fingerprint:  card.Fingerprint,
		}
	}
	return cards, nil
}

func (s *StripeService) AddCardForCustomer(cardToken, customerId string) (*common.Card, error) {
	params := url.Values{}
	params.Set("card", cardToken)

	customerCardEndpoint := fmt.Sprintf("%s/%s/cards", stripeCustomersUrl, customerId)
	sCard := &stripeCard{}
	if err := s.query("POST", customerCardEndpoint, params, sCard); err != nil {
		return nil, err
	}

	return &common.Card{
		ThirdPartyId: sCard.Id,
		Fingerprint:  sCard.Fingerprint,
	}, nil
}

func (s *StripeService) MakeCardDefaultForCustomer(cardId, customerId string) error {
	params := url.Values{}
	params.Set("default_card", cardId)

	customerUpdateEndpoint := fmt.Sprintf("%s/%s", stripeCustomersUrl, customerId)
	return s.query("POST", customerUpdateEndpoint, params, nil)
}

func (s *StripeService) DeleteCardForCustomer(customerId string, cardId string) error {
	deleteCustomerCardEndpoint := fmt.Sprintf("%s/%s/cards/%s", stripeCustomersUrl, customerId, cardId)
	return s.query("DELETE", deleteCustomerCardEndpoint, nil, nil)
}

func (s *StripeService) query(httpVerb, endPointUrl string, parameters url.Values, res interface{}) error {
	endPoint, err := url.Parse(endPointUrl)
	if err != nil {
		return err
	}

	endPoint.User = url.User(s.SecretKey)

	var body io.Reader
	if parameters != nil {
		switch httpVerb {
		case "GET", "DELETE":
			endPoint.RawQuery = parameters.Encode()
		case "POST":
			body = strings.NewReader(parameters.Encode())
		}
	}

	request, err := http.NewRequest(httpVerb, endPoint.String(), body)
	if err != nil {
		return err
	}
	request.Header.Set("Stripe-Version", apiVersion)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sError := &StripeError{}
		if err := json.NewDecoder(resp.Body).Decode(sError); err != nil {
			return err
		}
		return sError
	} else if res != nil {
		if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
			return err
		}
	}

	return nil
}

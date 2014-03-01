package stripe

import (
	"carefront/common"
	"carefront/libs/payment"
	"encoding/json"
	"fmt"
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

type stripeError struct {
	Code    int
	Details struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (s *StripeService) CreateCustomerWithDefaultCard(token string) (*payment.Customer, error) {
	params := url.Values{}
	params.Set("card", token)

	resp, err := s.query("POST", stripeCustomersUrl, params)
	if err != nil {
		return nil, err
	}

	sCustomer := &stripeCustomer{}
	if err := json.NewDecoder(resp.Body).Decode(sCustomer); err != nil {
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

func (s *StripeService) GetCardsForCustomer(customerId string) ([]common.Card, error) {
	resp, err := s.query("GET", fmt.Sprintf("%s/%s/cards", stripeCustomersUrl, customerId), nil)
	if err != nil {
		return nil, err
	}

	sCardData := &stripeCardData{}
	if err := json.NewDecoder(resp.Body).Decode(sCardData); err != nil {
		return nil, err
	}

	cards := make([]common.Card, len(sCardData.Data))
	for i, card := range sCardData.Data {
		cards[i] = common.Card{
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

func (s *StripeService) AddCardForCustomer(cardToken string, customerId string) (*common.Card, error) {
	params := url.Values{}
	params.Set("card", cardToken)

	customerCardEndpoint := fmt.Sprintf("%s/%s/cards", stripeCustomersUrl, customerId)
	resp, err := s.query("POST", customerCardEndpoint, params)
	if err != nil {
		return nil, err
	}

	sCard := &stripeCard{}
	if err := json.NewDecoder(resp.Body).Decode(sCard); err != nil {
		return nil, err
	}

	return &common.Card{
		ThirdPartyId: sCard.Id,
		Fingerprint:  sCard.Fingerprint,
	}, nil
}

func (s *StripeService) MakeCardDefaultForCustomer(cardId string, customerId string) error {
	params := url.Values{}
	params.Set("default_card", cardId)

	customerUpdateEndpoint := fmt.Sprintf("%s/%s", stripeCustomersUrl, customerId)
	_, err := s.query("POST", customerUpdateEndpoint, params)
	return err
}

func (s *StripeService) DeleteCardForCustomer(customerId string, cardId string) error {
	deleteCustomerCardEndpoint := fmt.Sprintf("%s/%s/cards/%s", stripeCustomersUrl, customerId, cardId)
	_, err := s.query("DELETE", deleteCustomerCardEndpoint, nil)
	return err
}

func (s *StripeService) query(httpVerb string, endPointUrl string, parameters url.Values) (*http.Response, error) {
	endPoint, err := url.Parse(endPointUrl)
	if err != nil {
		return nil, err
	}

	endPoint.User = url.User(s.SecretKey)

	var body *io.Reader
	if parameters != nil {
		switch httpVerb {
		case "GET":
			endPoint.RawQuery = parameters.Encode()
		case "POST", "DELETE":
			body = strings.NewReader(parameters.Encode())
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	request, err := http.NewRequest(httpVerb, endPoint.String(), body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Stripe-Version", apiVersion)
	resp, err := http.DefaultClient.Do(request)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		sError := &stripeError{}
		if err := json.NewDecoder(resp.Body).Decode(sError); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Something went wrong when making call to stripe %+v", sError)
	}

	return resp, err
}

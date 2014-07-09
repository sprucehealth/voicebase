package stripe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/payment"
)

const (
	apiURL        = "https://api.stripe.com/v1/"
	customersURL  = apiURL + "customers"
	recipientsURL = apiURL + "recipients"
	transfersURL  = apiURL + "transfers"
	apiVersion    = "2014-01-31"
)

type RecipientType string

const (
	Individual  RecipientType = "individual"
	Corporation RecipientType = "corporation"
)

func (t RecipientType) MarshalText() ([]byte, error) {
	return []byte(t), nil
}

func (t *RecipientType) UnmarshalText(text []byte) error {
	if text == nil {
		return nil
	}
	switch m := RecipientType(text); m {
	case Individual, Corporation:
		*t = m
	default:
		return fmt.Errorf("stripe: unknown recipient type %s", m)
	}
	return nil
}

func (t RecipientType) String() string {
	return string(t)
}

type StripeService struct {
	SecretKey      string
	PublishableKey string
}

type stripeCustomer struct {
	ID    string         `json:"id"`
	Cards stripeCardData `json:"cards"`
}

type stripeCard struct {
	ID          string `json:"id"`
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

type Recipient struct {
	ID            string            `json:"id"`
	Object        string            `json:"object"` // "recipient"
	Created       Timestamp         `json:"created"`
	LiveMode      bool              `json:"livemode"`
	Type          RecipientType     `json:"type"`
	Description   string            `json:"description"`
	Email         string            `json:"email"`
	Name          string            `json:"name"`
	Verified      bool              `json:"verified"`
	ActiveAccount *Account          `json:"active_account"`
	Metadata      map[string]string `json:"metadata"`
	// cards: {}
	// default_Card
}

type Account struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	BankName    string `json:"bank_name"`
	Last4       string `json:"last4"`
	Country     string `json:"country"`
	Currency    string `json:"currency"`
	Validated   bool   `json:"validated"`
	Verified    bool   `json:"verified"`
	Fingerprint string `json:"fingerprint"`
	Disabled    bool   `json:"disabled"`
}

type CreateRecipientRequest struct {
	Name             string        // required
	Type             RecipientType // required
	TaxID            string        // optional
	BankAccountToken string        // optional
	BankAccount      *BankAccount  // optional
	CardToken        string        // optional
	// TODO: Card *Card
	Email       string            // optional
	Description string            // optional
	Metadata    map[string]string // optional
}

type BankAccount struct {
	Country       string `json:"country"`
	RoutingNumber string `json:"routing_number"`
	AccountNumber string `json:"account_number"`
}

func (s *StripeError) Error() string {
	return fmt.Sprintf("Error communicating with stripe. ErrorCode: %dErrorDetails:\n- Type: %s\n- Message: %s\n- Code:%s\n", s.Code, s.Details.Type, s.Details.Message, s.Details.Code)
}

func (s *StripeService) CreateCustomerWithDefaultCard(token string) (*payment.Customer, error) {
	params := url.Values{}
	params.Set("card", token)

	sCustomer := &stripeCustomer{}
	if err := s.query("POST", customersURL, params, sCustomer); err != nil {
		return nil, err
	}

	if sCustomer.Cards.Count == 0 {
		return nil, fmt.Errorf("Expected atleast 1 card to be returned when creating the customer")
	}

	return &payment.Customer{
		Id: sCustomer.ID,
		Cards: []common.Card{
			common.Card{
				ThirdPartyId: sCustomer.Cards.Data[0].ID,
				Fingerprint:  sCustomer.Cards.Data[0].Fingerprint,
			},
		},
	}, nil
}

func (s *StripeService) GetCardsForCustomer(customerId string) ([]*common.Card, error) {
	sCardData := &stripeCardData{}
	if err := s.query("GET", fmt.Sprintf("%s/%s/cards", customersURL, customerId), nil, sCardData); err != nil {
		return nil, err
	}

	cards := make([]*common.Card, len(sCardData.Data))
	for i, card := range sCardData.Data {
		cards[i] = &common.Card{
			ThirdPartyId: card.ID,
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

	customerCardEndpoint := fmt.Sprintf("%s/%s/cards", customersURL, customerId)
	sCard := &stripeCard{}
	if err := s.query("POST", customerCardEndpoint, params, sCard); err != nil {
		return nil, err
	}

	return &common.Card{
		ThirdPartyId: sCard.ID,
		Fingerprint:  sCard.Fingerprint,
	}, nil
}

func (s *StripeService) MakeCardDefaultForCustomer(cardId, customerId string) error {
	params := url.Values{}
	params.Set("default_card", cardId)

	customerUpdateEndpoint := fmt.Sprintf("%s/%s", customersURL, customerId)
	return s.query("POST", customerUpdateEndpoint, params, nil)
}

func (s *StripeService) DeleteCardForCustomer(customerId string, cardId string) error {
	deleteCustomerCardEndpoint := fmt.Sprintf("%s/%s/cards/%s", customersURL, customerId, cardId)
	return s.query("DELETE", deleteCustomerCardEndpoint, nil, nil)
}

func (s *StripeService) CreateRecipient(req *CreateRecipientRequest) (*Recipient, error) {
	params := url.Values{}
	params.Set("name", req.Name)
	params.Set("type", string(req.Type))
	if req.TaxID != "" {
		params.Set("tax_id", req.TaxID)
	}
	if req.BankAccountToken != "" {
		params.Set("bank_account", req.BankAccountToken)
	} else if req.BankAccount != nil {
		params.Set("bank_account[country]", req.BankAccount.Country)
		params.Set("bank_account[routing_number]", req.BankAccount.RoutingNumber)
		params.Set("bank_account[account_number]", req.BankAccount.AccountNumber)
	}
	if req.CardToken != "" {
		params.Set("card", req.CardToken)
	}
	if req.Email != "" {
		params.Set("email", req.Email)
	}
	if req.Description != "description" {
		params.Set("description", req.Description)
	}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			params.Set(fmt.Sprintf("metadata[%s]", k), v)
		}
	}

	var recipient Recipient
	if err := s.query("POST", recipientsURL, params, &recipient); err != nil {
		return nil, err
	}

	return &recipient, nil
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

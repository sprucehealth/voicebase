package patient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/stripe"
)

type mockDataAPI_replaceCard struct {
	api.DataAPI
	patient      *common.Patient
	cards        []*common.Card
	cardToReturn *common.Card

	cardAdded   *common.Card
	cardDeleted *common.Card
}

func (m *mockDataAPI_replaceCard) GetPatientFromAccountID(id int64) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPI_replaceCard) GetCardsForPatient(id int64) ([]*common.Card, error) {
	return m.cards, nil
}
func (m *mockDataAPI_replaceCard) CreatePendingTask(workType, status string, itemID int64) (int64, error) {
	return 0, nil
}
func (m *mockDataAPI_replaceCard) UpdatePatient(id int64, update *api.PatientUpdate, doctorUpdate bool) error {
	return nil
}
func (m *mockDataAPI_replaceCard) AddCardForPatient(id int64, card *common.Card) error {
	m.cardAdded = card
	return nil
}
func (m *mockDataAPI_replaceCard) MakeCardDefaultForCustomer(thirdPartyID, customerID string) error {
	return nil
}
func (m *mockDataAPI_replaceCard) DeletePendingTask(id int64) error {
	return nil
}
func (m *mockDataAPI_replaceCard) MarkCardInactiveForPatient(patientID int64, card *common.Card) error {
	return nil
}
func (m *mockDataAPI_replaceCard) DeleteCardForPatient(id int64, card *common.Card) error {
	m.cardDeleted = card
	return nil
}
func (m *mockDataAPI_replaceCard) GetCardFromID(id int64) (*common.Card, error) {
	return m.cardToReturn, nil
}

type mockPaymentAPI_replaceCard struct {
	apiservice.StripeClient
	customer   *stripe.Customer
	stripeCard *stripe.Card

	deletedCardID  string
	addedCardToken string
	customerAdded  bool
}

func (m *mockPaymentAPI_replaceCard) CreateCustomerWithDefaultCard(token string) (*stripe.Customer, error) {
	m.addedCardToken = token
	m.customerAdded = true
	return m.customer, nil
}
func (m *mockPaymentAPI_replaceCard) AddCardForCustomer(token, customerID string) (*stripe.Card, error) {
	m.addedCardToken = token
	return m.stripeCard, nil
}
func (m *mockPaymentAPI_replaceCard) MakeCardDefaultForCustomer(thirdPartyID, customerID string) error {
	return nil
}
func (m *mockPaymentAPI_replaceCard) DeleteCardForCustomer(customerID, thirdPartyID string) error {
	m.deletedCardID = thirdPartyID
	return nil
}

func TestReplaceCard_NoCardExists(t *testing.T) {
	w := httptest.NewRecorder()

	jsonData, err := json.Marshal(replaceCardRequestData{
		Card: &common.Card{
			Token: "1234",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("PUT", "replace_card", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")

	m := &mockDataAPI_replaceCard{
		patient: &common.Patient{},
	}

	p := &mockPaymentAPI_replaceCard{
		customer: &stripe.Customer{
			ID: "12345",
			CardList: &stripe.CardList{
				Cards: []*stripe.Card{
					{},
				},
			},
		},
	}

	h := replaceCardHandler{
		dataAPI:    m,
		paymentAPI: p,
	}

	h.ServeHTTP(w, r)

	// ensure that customerID was set
	if m.patient.PaymentCustomerID != "12345" {
		t.Fatal("Expected customer id to be set for patient")
	}

	// ensure that card was added
	if m.cardAdded == nil {
		t.Fatal("Expected card to have been added but it wasnt")
	}

	// ensure card is default
	if !m.cardAdded.IsDefault {
		t.Fatal("Card added is not default")
	}

	// ensure expected card was added to payment service
	if p.addedCardToken != "1234" {
		t.Fatal("expected card was not added to payment service")
	}

	// ensure customer was added
	if !p.customerAdded {
		t.Fatal("customer expected to be added but wasnt")
	}
}

func TestReplaceCard_CardExists(t *testing.T) {
	w := httptest.NewRecorder()

	jsonData, err := json.Marshal(replaceCardRequestData{
		Card: &common.Card{
			Token: "1234",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("PUT", "replace_card", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")

	cardToReturn := &common.Card{
		ID:           encoding.NewObjectID(10),
		ThirdPartyID: "222",
	}
	m := &mockDataAPI_replaceCard{
		patient: &common.Patient{
			PaymentCustomerID: "0000",
			PatientAddress:    &common.Address{},
		},
		cards: []*common.Card{
			cardToReturn,
		},
		cardToReturn: cardToReturn,
	}

	p := &mockPaymentAPI_replaceCard{
		stripeCard: &stripe.Card{
			ID: "4567",
		},
	}

	h := replaceCardHandler{
		dataAPI:    m,
		paymentAPI: p,
	}

	h.ServeHTTP(w, r)

	// ensure that card was added
	if m.cardAdded == nil {
		t.Fatal("Expected card to have been added but it wasnt")
	}

	// ensure card is default
	if !m.cardAdded.IsDefault {
		t.Fatal("Card added is not default")
	}

	// ensure card added has id as expceted
	if m.cardAdded.ThirdPartyID != "4567" {
		t.Fatal("Card added doesn't have expected id")
	}

	// ensure card was deleted
	if m.cardDeleted == nil {
		t.Fatal("expected card to be deleted but it wasnt")
	}

	// ensure card deleted had the expected id
	if m.cardDeleted.ID.Int64() != int64(10) {
		t.Fatal("expected card was not deleted")
	}

	// ensure that payment service was informed of deleted card
	if p.deletedCardID != "222" {
		t.Fatal("epected card was not deleted from payment service")
	}

	// ensure expected card was added to payment service
	if p.addedCardToken != "1234" {
		t.Fatal("expected card was not added to payment service")
	}

}

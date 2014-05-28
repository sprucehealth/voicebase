package test_integration

import (
	"bytes"
	"carefront/address"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/payment"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAddCardsForPatient(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	customerToAdd := &payment.Customer{
		Id: "test_customer_id",
		Cards: []common.Card{common.Card{
			ThirdPartyId: "third_party_id0",
			Fingerprint:  "test_fingerprint0",
		},
		},
	}

	stubPaymentsService := &payment.StubPaymentService{
		CustomerToReturn: customerToAdd,
	}

	stubAddressValidationService := &address.StubAddressValidationService{
		CityStateToReturn: address.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}

	patientCardsHandler := &apiservice.PatientCardsHandler{
		DataApi:              testData.DataApi,
		PaymentApi:           stubPaymentsService,
		AddressValidationApi: stubAddressValidationService,
	}

	ts := httptest.NewServer(patientCardsHandler)
	defer ts.Close()

	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	card1, localCards := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService, nil)

	// check to ensure there is no pending task left
	checkPendingTaskCount(t, testData, patient.PatientId.Int64())

	//  get the patient address to see what the address is
	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	checkBillingAddress(t, patient, card1.BillingAddress)

	if len(localCards) != 1 {
		t.Fatalf("Expected to get back one card saved for patient instead got back %d", len(localCards))
	}

	if localCards[0].ThirdPartyId != customerToAdd.Cards[0].ThirdPartyId ||
		localCards[0].Fingerprint != customerToAdd.Cards[0].Fingerprint ||
		!localCards[0].IsDefault {
		t.Fatalf("card added has different id and finger print than was expected")
	}

	card2, localCards := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService, localCards)

	checkPendingTaskCount(t, testData, patient.PatientId.Int64())
	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	checkBillingAddress(t, patient, card2.BillingAddress)

	if len(localCards) != 2 {
		t.Fatalf("Expected to get back 2 cards saved for patient instead got back %d", len(localCards))
	}

	defaultCardFound := false
	for _, localCard := range localCards {
		if localCard.IsDefault {
			if localCard.ThirdPartyId != stubPaymentsService.CardToReturnOnAdd.ThirdPartyId {
				t.Fatal("Expected the card just added to be the default card but it wasnt")
			}
			defaultCardFound = true
		}
	}

	if !defaultCardFound {
		t.Fatalf("Expected one of the cards to be the default but none were")
	}

	// now, lets try to make the previous card the default again
	var cardToMakeDefault *common.Card
	for _, localCard := range localCards {
		if localCard.ThirdPartyId != stubPaymentsService.CardToReturnOnAdd.ThirdPartyId {
			cardToMakeDefault = localCard
		}
	}

	params := url.Values{}
	params.Set("card_id", strconv.FormatInt(cardToMakeDefault.Id.Int64(), 10))
	resp, err := AuthPut(ts.URL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()),
		patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make previous card default: " + err.Error())
	}

	patientCardsResponse := &apiservice.PatientCardsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(patientCardsResponse); err != nil {
		t.Fatalf("Unable to unmarshal response body into patient cards response: %+v", err)
	}
	localCards = patientCardsResponse.Cards

	CheckSuccessfulStatusCode(resp, "Unable to make previous card default", t)

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	checkBillingAddress(t, patient, card1.BillingAddress)

	if len(localCards) != 2 {
		t.Fatalf("Expected to get back 2 cards saved for patient instead got back %d", len(localCards))
	}

	defaultCardFound = false
	for _, localCard := range localCards {
		if localCard.IsDefault {
			if localCard.ThirdPartyId != cardToMakeDefault.ThirdPartyId {
				t.Fatal("Expected the card just made default to be the default card but it wasnt")
			}
			defaultCardFound = true
		}
	}

	if !defaultCardFound {
		t.Fatalf("Expected one of the cards to be the default but none where")
	}

	// lets delete the default card
	localCards = deleteCard(t, testData, patient, cardToMakeDefault, stubPaymentsService, patientCardsHandler, localCards)

	if !localCards[0].IsDefault {
		t.Fatal("Expected the only remaining card to be the default")
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id " + err.Error())
	}

	checkBillingAddress(t, patient, card2.BillingAddress)

	// lets delete the last card
	localCards = deleteCard(t, testData, patient, localCards[0], stubPaymentsService, patientCardsHandler, localCards)

	if len(localCards) != 0 {
		t.Fatalf("expected to be 0 cards but there was %d ", len(localCards))
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	checkBillingAddress(t, patient, card2.BillingAddress)

	// sleep between adding of cards so that we can consistently ensure that the right card was made default
	// this is because mysql does not support millisecond/microsecond level precision unless specified, and
	// this would make it so that if cards were added within the second, we could not consistently say
	// which card was made default
	card3, localCards := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService, localCards)
	time.Sleep(time.Second)
	card4, localCards := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService, localCards)
	time.Sleep(time.Second)
	card5, localCards := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService, localCards)

	// the cards should appear in ascending order of being added.
	if localCards[0].ThirdPartyId != card3.ThirdPartyId {
		t.Fatal("Expected the first card returned to be card3 but it wasnt")
	}

	if localCards[1].ThirdPartyId != card4.ThirdPartyId {
		t.Fatal("Expected the second card returned to be card4 but it wasnt")
	}

	if localCards[2].ThirdPartyId != card5.ThirdPartyId {
		t.Fatal("Expected the third card returned to be card5 but it wasn't")
	}

	defaultCardFound = false
	for _, localCard := range localCards {

		if localCard.IsDefault {
			if localCard.ThirdPartyId != card5.ThirdPartyId {
				t.Fatal("Expected card5 to be the default given it was the last carded added")
			}
			defaultCardFound = true
		}

	}

	if !defaultCardFound {
		t.Fatal("No default card found when card5 expected to be default")
	}

	var cardToDelete *common.Card
	for _, localCard := range localCards {
		if localCard.IsDefault {
			cardToDelete = localCard
			break
		}
	}

	if cardToDelete == nil {
		t.Fatal("Unable to find the card to delete")
	}

	// delete card 5 and card 4 should be the default card
	localCards = deleteCard(t, testData, patient, cardToDelete, stubPaymentsService, patientCardsHandler, localCards)

	if len(localCards) != 2 {
		t.Fatalf("Expected to get 2 cards but instead got %d", len(localCards))
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient for id ", err.Error())
	}

	// identify card3 as the next card to be deleted (which is not the default) while
	// also checking to make sure that card4 is actually the current default
	cardToDelete = nil
	for _, localCard := range localCards {
		if localCard.IsDefault {
			if localCard.ThirdPartyId != card4.ThirdPartyId {
				t.Fatalf("Expected the 4th card to be the default card but it wasnt. Local Card thirdpartyId: %s, card 4 thirdpartyid: %s", localCard.ThirdPartyId, card4.ThirdPartyId)
			}
		} else {
			if localCard.ThirdPartyId == card3.ThirdPartyId {
				cardToDelete = localCard
			}
		}
	}
	checkBillingAddress(t, patient, card4.BillingAddress)

	if cardToDelete == nil {
		t.Fatal("Unable to locate card3 which should exist")
	}

	localCards = deleteCard(t, testData, patient, cardToDelete, stubPaymentsService, patientCardsHandler, localCards)

	if len(localCards) != 1 {
		t.Fatalf("Expected 1 card to remain instead got back %d", len(localCards))
	}

	if localCards[0].ThirdPartyId != card4.ThirdPartyId {
		t.Fatalf("Expected card4 (%s) to be returned instead got back %s", card4.ThirdPartyId, localCards[0].ThirdPartyId)
	}

	if !localCards[0].IsDefault {
		t.Fatal("Expected the single remaining card to be the default")
	}
}

func deleteCard(t *testing.T, TestData TestData, patient *common.Patient, cardToDelete *common.Card, stubPaymentsService *payment.StubPaymentService, patientCardsHandler *apiservice.PatientCardsHandler, currentCards []*common.Card) []*common.Card {
	params := url.Values{}
	params.Set("card_id", strconv.FormatInt(cardToDelete.Id.Int64(), 10))

	updatedCards := make([]*common.Card, 0)
	for _, card := range currentCards {
		if card.ThirdPartyId != cardToDelete.ThirdPartyId {
			updatedCards = append(updatedCards, card)
		}
	}
	stubPaymentsService.CardsToReturn = updatedCards

	ts := httptest.NewServer(patientCardsHandler)
	defer ts.Close()

	resp, err := AuthDelete(ts.URL+"?"+params.Encode(), "application/x-www-form-urlencoded", nil, patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to delete card: " + err.Error())
	}

	patientCardsResponse := &apiservice.PatientCardsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(patientCardsResponse); err != nil {
		t.Fatalf("Unable to unmarshal cards ")
	}

	CheckSuccessfulStatusCode(resp, "Unable to delete card", t)

	return patientCardsResponse.Cards
}

func addCard(t *testing.T, testData TestData, patientAccountId int64, patientCardsHandler *apiservice.PatientCardsHandler, stubPaymentsService *payment.StubPaymentService, currentCards []*common.Card) (*common.Card, []*common.Card) {

	ts := httptest.NewServer(patientCardsHandler)
	defer ts.Close()

	billingAddress := &common.Address{
		AddressLine1: "1234 Main Street " + strconv.FormatInt(time.Now().UnixNano(), 10),
		AddressLine2: "Apt 12345",
		City:         "San Francisco",
		State:        "California",
		ZipCode:      "12345",
	}

	card := &common.Card{
		Token:          "1235 " + strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:           "Visa",
		BillingAddress: billingAddress,
	}

	stubPaymentsService.CardToReturnOnAdd = &common.Card{
		Fingerprint:  fmt.Sprintf("test_fingerprint%d", len(currentCards)),
		ThirdPartyId: fmt.Sprintf("third_party_id%d", len(currentCards)),
	}

	card.ThirdPartyId = stubPaymentsService.CardToReturnOnAdd.ThirdPartyId
	card.Fingerprint = stubPaymentsService.CardToReturnOnAdd.Fingerprint

	stubPaymentsService.CardsToReturn = make([]*common.Card, 0)
	// lets add the cards out of order to ensure they come back in ascending order
	stubPaymentsService.CardsToReturn = append(stubPaymentsService.CardsToReturn, card)
	if currentCards != nil {
		stubPaymentsService.CardsToReturn = append(stubPaymentsService.CardsToReturn, currentCards...)
	}

	jsonData, err := json.Marshal(card)

	if err != nil {
		t.Fatal("Unable to marshal card object: " + err.Error())
	}

	resp, err := AuthPost(ts.URL, "application/json", bytes.NewReader(jsonData), patientAccountId)
	if err != nil {
		t.Fatal("Unable to make successful call to add cards to patient: " + err.Error())
	}

	if err != nil {
		t.Fatal("Unable to successfully add card to customer " + err.Error())
	}

	patientCardsResponse := &apiservice.PatientCardsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(patientCardsResponse); err != nil {
		t.Fatalf("Unable to unmarshal response body into cardsResponse object: %+v", err)
	}

	CheckSuccessfulStatusCode(resp, "Unable to add card to patient", t)

	return card, patientCardsResponse.Cards
}

func checkPendingTaskCount(t *testing.T, testData TestData, patientId int64) {
	var pendingTaskCount int64
	err := testData.DB.QueryRow(`select count(*) from pending_task where item_id = ?`, patientId).Scan(&pendingTaskCount)
	if err != nil {
		t.Fatal("Unable to count the number of pending tasks remaining: " + err.Error())
	}

	if pendingTaskCount != 0 {
		t.Fatal("Expected there to remain no pending tasks once the card was added for a patient")
	}
}

func checkBillingAddress(t *testing.T, patient *common.Patient, addressToCompare *common.Address) {
	if patient.PatientAddress == nil ||
		patient.PatientAddress.AddressLine1 != addressToCompare.AddressLine1 ||
		patient.PatientAddress.AddressLine2 != addressToCompare.AddressLine2 ||
		patient.PatientAddress.City != addressToCompare.City ||
		patient.PatientAddress.State != addressToCompare.State ||
		patient.PatientAddress.ZipCode != addressToCompare.ZipCode {
		t.Fatal("Default address for the patient doesn't match the billing address as it should")
	}
}

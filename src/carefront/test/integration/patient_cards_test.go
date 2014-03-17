package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/payment"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAddCardsForPatient(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	customerToAdd := &payment.Customer{
		Id: "test_customer_id",
		Cards: []common.Card{common.Card{
			ThirdPartyId: "test_card_id",
			Fingerprint:  "test_finger_print",
		},
		},
	}

	stubPaymentsService := &payment.StubPaymentService{
		CustomerToReturn: customerToAdd,
	}

	patientCardsHandler := &apiservice.PatientCardsHandler{
		DataApi:    testData.DataApi,
		PaymentApi: stubPaymentsService,
	}

	ts := httptest.NewServer(patientCardsHandler)
	defer ts.Close()

	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	card1 := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService)

	// check to ensure there is no pending task left
	checkPendingTaskCount(t, testData, patient.PatientId.Int64())

	//  get the patient address to see what the address is
	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	checkBillingAddress(t, patient, card1.BillingAddress)

	//  Get cards with the patient to ensure that they have just one and this is the one
	localCards, err := testData.DataApi.GetCardsForPatient(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get cards for patient: " + err.Error())
	}

	if len(localCards) != 1 {
		t.Fatalf("Expected to get back one card saved for patient instead got back %d", len(localCards))
	}

	if localCards[0].ThirdPartyId != customerToAdd.Cards[0].ThirdPartyId ||
		localCards[0].Fingerprint != customerToAdd.Cards[0].Fingerprint ||
		!localCards[0].IsDefault {
		t.Fatalf("card added has different id and finger print than was expected")
	}

	card2 := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService)

	checkPendingTaskCount(t, testData, patient.PatientId.Int64())
	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	checkBillingAddress(t, patient, card2.BillingAddress)

	localCards, err = testData.DataApi.GetCardsForPatient(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get cards for patient: " + err.Error())
	}

	if len(localCards) != 2 {
		t.Fatalf("Expected to get back 2 cards saved for patient instead got back %d", len(localCards))
	}

	defaultCardFound := false
	for _, localCard := range localCards {
		if localCard.ThirdPartyId == stubPaymentsService.CardToReturnOnAdd.ThirdPartyId {
			if !localCard.IsDefault {
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
	resp, err := authPut(ts.URL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()),
		patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make previous card default: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make previous card default", t)

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	checkBillingAddress(t, patient, card1.BillingAddress)

	localCards, err = testData.DataApi.GetCardsForPatient(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get cards for patient: " + err.Error())
	}

	if len(localCards) != 2 {
		t.Fatalf("Expected to get back 2 cards saved for patient instead got back %d", len(localCards))
	}

	defaultCardFound = false
	for _, localCard := range localCards {
		if localCard.ThirdPartyId == cardToMakeDefault.ThirdPartyId {
			if !localCard.IsDefault {
				t.Fatal("Expected the card just made default to be the default card but it wasnt")
			}
			defaultCardFound = true
		}
	}

	if !defaultCardFound {
		t.Fatalf("Expected one of the cards to be the default but none where")
	}

	// lets delete the default card
	params = url.Values{}
	params.Set("card_id", strconv.FormatInt(cardToMakeDefault.Id.Int64(), 10))
	resp, err = authDelete(ts.URL+"?"+params.Encode(), "application/x-www-form-urlencoded", nil, patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to delete card: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to delete card", t)

	localCards, err = testData.DataApi.GetCardsForPatient(patient.PatientId.Int64())
	if len(localCards) != 1 {
		t.Fatalf("Expected just 1 card after the card was deleted instead got %d", len(localCards))
	}

	if !localCards[0].IsDefault {
		t.Fatal("Expected the only remaining card to be the default")
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id " + err.Error())
	}

	checkBillingAddress(t, patient, card2.BillingAddress)

	// lets delete the last card
	params = url.Values{}
	params.Set("card_id", strconv.FormatInt(localCards[0].Id.Int64(), 10))
	resp, err = authDelete(ts.URL+"?"+params.Encode(), "application/x-www-form-urlencoded", nil, patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to delete card: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to delete card", t)

	localCards, err = testData.DataApi.GetCardsForPatient(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to delete card: " + err.Error())
	}

	if len(localCards) != 0 {
		t.Fatalf("expected to be 0 cards but there was %d ", len(localCards))
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	checkBillingAddress(t, patient, card2.BillingAddress)

	card3 := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService)
	t.Logf("Card 3 added with third party id : %s", card3.ThirdPartyId)
	card4 := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService)
	t.Logf("Card 4 added with third party id : %s", card4.ThirdPartyId)
	card5 := addCard(t, testData, patient.AccountId.Int64(), patientCardsHandler, stubPaymentsService)
	t.Logf("Card 5 added with third party id : %s", card5.ThirdPartyId)

	localCards, err = testData.DataApi.GetCardsForPatient(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to delete card: " + err.Error())
	}

	defaultCardFound = false
	for _, localCard := range localCards {
		if localCard.ThirdPartyId == card5.ThirdPartyId {
			if !localCard.IsDefault {
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
	params = url.Values{}
	params.Set("card_id", strconv.FormatInt(cardToDelete.Id.Int64(), 10))
	resp, err = authDelete(ts.URL+"?"+params.Encode(), "application/x-www-form-urlencoded", nil, patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to delete card: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to delete card", t)

	localCards, err = testData.DataApi.GetCardsForPatient(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get local cards " + err.Error())
	}

	if len(localCards) != 2 {
		t.Fatalf("Expected to get 2 cards but instead got %d", len(localCards))
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient for id ", err.Error())
	}

	for _, localCard := range localCards {
		if localCard.IsDefault {
			if localCard.ThirdPartyId != card4.ThirdPartyId {
				t.Fatalf("Expected the 4th card to be the default card but it wasnt. Local Card thirdpartyId: %s, card 4 thirdpartyid: %s", localCard.ThirdPartyId, card4.ThirdPartyId)
			}
		}
	}

	checkBillingAddress(t, patient, card4.BillingAddress)
}

func addCard(t *testing.T, testData TestData, patientAccountId int64, patientCardsHandler *apiservice.PatientCardsHandler, stubPaymentsService *payment.StubPaymentService) *common.Card {

	ts := httptest.NewServer(patientCardsHandler)
	defer ts.Close()

	billingAddress := &common.Address{
		AddressLine1: "1234 Main Street " + strconv.FormatInt(time.Now().UnixNano(), 10),
		AddressLine2: "Apt 12345",
		City:         "San Francisco",
		State:        "CA",
	}

	card := &common.Card{
		Token:          "1235 " + strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:           "Visa",
		BillingAddress: billingAddress,
	}

	stubPaymentsService.CardToReturnOnAdd = &common.Card{
		Fingerprint:  "test_fingerprint" + strconv.FormatInt(time.Now().UnixNano(), 10),
		ThirdPartyId: "test_id " + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	card.ThirdPartyId = stubPaymentsService.CardToReturnOnAdd.ThirdPartyId

	jsonData, err := json.Marshal(card)

	if err != nil {
		t.Fatal("Unable to marshal card object: " + err.Error())
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewReader(jsonData), patientAccountId)
	if err != nil {
		t.Fatal("Unable to make successful call to add cards to patient: " + err.Error())
	}

	if err != nil {
		t.Fatal("Unable to successfully add card to customer " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to add card to patient", t)

	return card
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

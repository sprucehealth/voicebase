package test_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/stripe"
	patientpkg "github.com/sprucehealth/backend/patient"
)

func TestAddCardsForPatient(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	signedupPatientResponse := SignupRandomTestPatient(t, testData)

	customerToAdd := &stripe.Customer{
		Id: "test_customer_id",
		CardList: &stripe.CardList{
			Cards: []*stripe.Card{&stripe.Card{
				ID:          "third_party_id0",
				Fingerprint: "test_fingerprint0",
			},
			},
		},
	}

	stubPaymentsService := testData.Config.PaymentAPI.(*StripeStub)
	stubPaymentsService.CustomerToReturn = customerToAdd

	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	card1, localCards := addCard(t, testData, patient.AccountId.Int64(), stubPaymentsService, nil)

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

	if localCards[0].ThirdPartyID != customerToAdd.CardList.Cards[0].ID ||
		localCards[0].Fingerprint != customerToAdd.CardList.Cards[0].Fingerprint ||
		!localCards[0].IsDefault {
		t.Fatalf("card added has different id and finger print than was expected")
	}

	card2, localCards := addCard(t, testData, patient.AccountId.Int64(), stubPaymentsService, localCards)

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
			if localCard.ThirdPartyID != stubPaymentsService.CardToReturnOnAdd.ID {
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
		if localCard.ThirdPartyID != stubPaymentsService.CardToReturnOnAdd.ID {
			cardToMakeDefault = localCard
		}
	}

	params := url.Values{}
	params.Set("card_id", strconv.FormatInt(cardToMakeDefault.ID.Int64(), 10))
	resp, err := testData.AuthPut(testData.APIServer.URL+router.PatientCardURLPath, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()),
		patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make previous card default: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d", resp.StatusCode)
	}

	patientCardsResponse := &patientpkg.PatientCardsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(patientCardsResponse); err != nil {
		t.Fatalf("Unable to unmarshal response body into patient cards response: %+v", err)
	}
	localCards = patientCardsResponse.Cards

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
			if localCard.ThirdPartyID != cardToMakeDefault.ThirdPartyID {
				t.Fatal("Expected the card just made default to be the default card but it wasnt")
			}
			defaultCardFound = true
		}
	}

	if !defaultCardFound {
		t.Fatalf("Expected one of the cards to be the default but none where")
	}

	// lets delete the default card
	localCards = deleteCard(t, testData, patient, cardToMakeDefault, stubPaymentsService, localCards)

	if !localCards[0].IsDefault {
		t.Fatal("Expected the only remaining card to be the default")
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id " + err.Error())
	}

	checkBillingAddress(t, patient, card2.BillingAddress)

	// lets delete the last card
	localCards = deleteCard(t, testData, patient, localCards[0], stubPaymentsService, localCards)

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
	card3, localCards := addCard(t, testData, patient.AccountId.Int64(), stubPaymentsService, localCards)
	time.Sleep(time.Second)
	card4, localCards := addCard(t, testData, patient.AccountId.Int64(), stubPaymentsService, localCards)
	time.Sleep(time.Second)
	card5, localCards := addCard(t, testData, patient.AccountId.Int64(), stubPaymentsService, localCards)

	// the cards should appear in ascending order of being added.
	if localCards[0].ThirdPartyID != card3.ThirdPartyID {
		t.Fatal("Expected the first card returned to be card3 but it wasnt")
	}

	if localCards[1].ThirdPartyID != card4.ThirdPartyID {
		t.Fatal("Expected the second card returned to be card4 but it wasnt")
	}

	if localCards[2].ThirdPartyID != card5.ThirdPartyID {
		t.Fatal("Expected the third card returned to be card5 but it wasn't")
	}

	defaultCardFound = false
	for _, localCard := range localCards {

		if localCard.IsDefault {
			if localCard.ThirdPartyID != card5.ThirdPartyID {
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
	localCards = deleteCard(t, testData, patient, cardToDelete, stubPaymentsService, localCards)

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
			if localCard.ThirdPartyID != card4.ThirdPartyID {
				t.Fatalf("Expected the 4th card to be the default card but it wasnt. Local Card thirdpartyId: %s, card 4 thirdpartyid: %s", localCard.ThirdPartyID, card4.ThirdPartyID)
			}
		} else {
			if localCard.ThirdPartyID == card3.ThirdPartyID {
				cardToDelete = localCard
			}
		}
	}
	checkBillingAddress(t, patient, card4.BillingAddress)

	if cardToDelete == nil {
		t.Fatal("Unable to locate card3 which should exist")
	}

	localCards = deleteCard(t, testData, patient, cardToDelete, stubPaymentsService, localCards)

	if len(localCards) != 1 {
		t.Fatalf("Expected 1 card to remain instead got back %d", len(localCards))
	}

	if localCards[0].ThirdPartyID != card4.ThirdPartyID {
		t.Fatalf("Expected card4 (%s) to be returned instead got back %s", card4.ThirdPartyID, localCards[0].ThirdPartyID)
	}

	if !localCards[0].IsDefault {
		t.Fatal("Expected the single remaining card to be the default")
	}
}

func deleteCard(t *testing.T, testData *TestData, patient *common.Patient, cardToDelete *common.Card, stripeStub *StripeStub, currentCards []*common.Card) []*common.Card {
	params := url.Values{}
	params.Set("card_id", strconv.FormatInt(cardToDelete.ID.Int64(), 10))

	updatedCards := make([]*stripe.Card, 0)
	for _, card := range currentCards {
		if card.ThirdPartyID != cardToDelete.ThirdPartyID {
			updatedCards = append(updatedCards, &stripe.Card{
				ID:          card.ThirdPartyID,
				Fingerprint: card.Fingerprint,
			})
		}
	}
	stripeStub.CardsToReturn = updatedCards

	resp, err := testData.AuthDelete(testData.APIServer.URL+router.PatientCardURLPath+"?"+params.Encode(), "application/x-www-form-urlencoded", nil, patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to delete card: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d instead", resp.StatusCode)
	}

	patientCardsResponse := &patientpkg.PatientCardsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(patientCardsResponse); err != nil {
		t.Fatalf("Unable to unmarshal cards ")
	}

	return patientCardsResponse.Cards
}

func addCard(t *testing.T, testData *TestData, patientAccountId int64, stripeStub *StripeStub, currentCards []*common.Card) (*common.Card, []*common.Card) {
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

	stripeStub.CardToReturnOnAdd = &stripe.Card{
		Fingerprint: fmt.Sprintf("test_fingerprint%d", len(currentCards)),
		ID:          fmt.Sprintf("third_party_id%d", len(currentCards)),
	}

	card.ThirdPartyID = stripeStub.CardToReturnOnAdd.ID
	card.Fingerprint = stripeStub.CardToReturnOnAdd.Fingerprint

	stripeStub.CardsToReturn = make([]*stripe.Card, 0)
	// lets add the cards out of order to ensure they come back in ascending order
	stripeStub.CardsToReturn = append(stripeStub.CardsToReturn, stripeStub.CardToReturnOnAdd)
	if currentCards != nil {
		for _, cCard := range currentCards {
			stripeStub.CardsToReturn = append(stripeStub.CardsToReturn, &stripe.Card{
				Fingerprint: cCard.Fingerprint,
				ID:          cCard.ThirdPartyID,
			})
		}
	}

	jsonData, err := json.Marshal(card)
	if err != nil {
		t.Fatal("Unable to marshal card object: " + err.Error())
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+router.PatientCardURLPath, "application/json", bytes.NewReader(jsonData), patientAccountId)
	if err != nil {
		t.Fatal("Unable to make successful call to add cards to patient: " + err.Error())
	}
	defer resp.Body.Close()

	if err != nil {
		t.Fatal("Unable to successfully add card to customer " + err.Error())
	}

	patientCardsResponse := &patientpkg.PatientCardsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(patientCardsResponse); err != nil {
		t.Fatalf("Unable to unmarshal response body into cardsResponse object: %+v", err)
	}

	return card, patientCardsResponse.Cards
}

func checkPendingTaskCount(t *testing.T, testData *TestData, patientId int64) {
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

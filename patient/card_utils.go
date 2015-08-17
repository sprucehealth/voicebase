package patient

import (
	"net/http"
	"sort"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/surescripts"
)

func deleteCard(
	cardID int64,
	patient *common.Patient,
	switchDefaultCard bool,
	dataAPI api.DataAPI,
	paymentAPI apiservice.StripeClient) error {

	card, err := dataAPI.GetCardFromID(cardID)
	if err != nil {
		return err
	}

	if card == nil {
		return apiservice.NewError("Card not found", http.StatusNotFound)
	}

	pendingTaskID, err := dataAPI.CreatePendingTask(api.PendingTaskPatientCard, api.StatusDeleting, patient.ID.Int64())
	if err != nil {
		return err
	}

	if patient.PaymentCustomerID == "" {
		return apiservice.NewError("Patient not registered yet for accepting payments", http.StatusBadRequest)
	}

	// mark the card as inactive instead of deleting it initially so that we have room to identify
	// situations where the call fails and things are left in an inconsistent state
	if err := dataAPI.MarkCardInactiveForPatient(patient.ID, card); err != nil {
		return err
	}

	// switch over the default card to the last added card if we are currently deleting the default card
	if card.IsDefault && switchDefaultCard {
		latestCard, err := dataAPI.MakeLatestCardDefaultForPatient(patient.ID)
		if err != nil {
			return err
		}

		if latestCard != nil {
			if err := dataAPI.UpdateDefaultAddressForPatient(patient.ID, latestCard.BillingAddress); err != nil {
				return err
			}
		}
	}

	// the payment service changes the default card to the last added active card internally
	if err := paymentAPI.DeleteCardForCustomer(patient.PaymentCustomerID, card.ThirdPartyID); err != nil {
		return err
	}

	if err := dataAPI.DeleteCardForPatient(patient.ID, card); err != nil {
		return err
	}

	var currentPatientAddressID int64
	if patient.PatientAddress != nil {
		currentPatientAddressID = patient.PatientAddress.ID
	}

	// delete the address only if this is not the patient's preferred address
	if card.BillingAddress != nil && currentPatientAddressID != card.BillingAddress.ID {
		if err := dataAPI.DeleteAddress(card.BillingAddress.ID); err != nil {
			return err
		}
	}

	if err := dataAPI.DeletePendingTask(pendingTaskID); err != nil {
		return err
	}

	return nil
}

func getCardsAndReconcileWithPaymentService(patient *common.Patient, dataAPI api.DataAPI, paymentAPI apiservice.StripeClient) ([]*common.Card, error) {
	localCards, err := dataAPI.GetCardsForPatient(patient.ID)
	if err != nil {
		return nil, err
	}

	if len(localCards) == 0 {
		return localCards, nil
	}

	stripeCards, err := paymentAPI.GetCardsForCustomer(patient.PaymentCustomerID)
	if err != nil {
		return nil, err
	}

	// log this fact so that we can figure out what is going on
	if len(localCards) != len(stripeCards) {
		golog.Warningf("Number of cards returned from payment service differs from number of cards locally stored for patient with id %d", patient.ID.Int64())
	}

	// trust the cards from the payment service as the source of authority
	var cards []*common.Card
	for _, cardFromService := range stripeCards {
		localCardFound := false
		card := &common.Card{
			Type:         cardFromService.Type,
			ThirdPartyID: cardFromService.ID,
			Fingerprint:  cardFromService.Fingerprint,
			ExpMonth:     cardFromService.ExpMonth,
			ExpYear:      cardFromService.ExpYear,
			Last4:        cardFromService.Last4,
		}
		for _, localCard := range localCards {
			if localCard.ThirdPartyID == cardFromService.ID {
				card.ID = localCard.ID
				card.IsDefault = localCard.IsDefault
				card.CreationDate = localCard.CreationDate
				card.ApplePay = localCard.ApplePay
				localCardFound = true
			}
		}
		if !localCardFound {
			golog.Warningf("Local card not found in set of cards returned from payment service for patient with id %d", patient.ID.Int64())
		}
		if !card.ApplePay {
			cards = append(cards, card)
		}
	}

	// sort cards by creation date so that customer seems them in the order that they entered the cards
	sort.Sort(common.ByCreationDate(cards))

	return cards, nil
}

func addCardForPatient(
	dataAPI api.DataAPI,
	paymentAPI apiservice.StripeClient,
	addressValidator address.Validator,
	cardToAdd *common.Card,
	patient *common.Patient,
	enforceAddressRequirement bool,
) error {

	if enforceAddressRequirement {
		if cardToAdd.BillingAddress == nil {
			return apiservice.NewError("Billing address for credit card not correctly specified", http.StatusBadRequest)
		}
		if err := cardToAdd.BillingAddress.Validate(); err != nil {
			return apiservice.NewError(err.Error(), http.StatusBadRequest)
		}
		if err := surescripts.ValidateAddress(
			cardToAdd.BillingAddress,
			addressValidator,
			dataAPI); err != nil {
			return err
		}
	}

	if cardToAdd.Token == "" {
		return apiservice.NewError(
			"Unable to add credit card that does not have a unique token to help identify the card with the third party service",
			http.StatusBadRequest)
	}

	// create a pending task to indicate that there's work that is currently in progress
	// to add a credit card for a patient. The reason to do this is to identify any tasks that span multiple steps
	// that may fail to complete half way through and then reconcile the work through a worker
	// that cleans things up
	pendingTaskID, err := dataAPI.CreatePendingTask(api.PendingTaskPatientCard, api.StatusCreating, patient.ID.Int64())
	if err != nil {
		return err
	}

	isPatientRegisteredWithPatientService := patient.PaymentCustomerID != ""
	var stripeCard *stripe.Card
	// if it does not exist, go ahead and create one with in stripe
	if !isPatientRegisteredWithPatientService {
		customer, err := paymentAPI.CreateCustomerWithDefaultCard(cardToAdd.Token)
		if err != nil {
			return err
		}

		// save customer id to database
		customerID := customer.ID
		if err := dataAPI.UpdatePatient(patient.ID, &api.PatientUpdate{
			StripeCustomerID: &customerID,
		}, false); err != nil {
			return err
		}
		stripeCard = customer.CardList.Cards[0]
		patient.PaymentCustomerID = customer.ID
	} else {
		// add another card to the customer on the payment service
		stripeCard, err = paymentAPI.AddCardForCustomer(cardToAdd.Token, patient.PaymentCustomerID)
		if err != nil {
			return err
		}
	}

	cardToAdd.ThirdPartyID = stripeCard.ID
	cardToAdd.Fingerprint = stripeCard.Fingerprint
	if err := dataAPI.AddCardForPatient(patient.ID, cardToAdd); err != nil {
		return err
	}

	// the card added for an existing patient does not become default on add; need to explicitly make a call to stripe
	// to make it the default card
	if isPatientRegisteredWithPatientService {
		if err := paymentAPI.MakeCardDefaultForCustomer(cardToAdd.ThirdPartyID, patient.PaymentCustomerID); err != nil {
			return err
		}
	}

	if cardToAdd.BillingAddress != nil {
		if err := dataAPI.UpdateDefaultAddressForPatient(patient.ID, cardToAdd.BillingAddress); err != nil {
			return err
		}
	}

	if err := dataAPI.DeletePendingTask(pendingTaskID); err != nil {
		return err
	}

	return nil
}

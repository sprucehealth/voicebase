package patient

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/surescripts"
)

type cardsHandler struct {
	dataAPI              api.DataAPI
	paymentAPI           apiservice.StripeClient
	addressValidationAPI address.Validator
}

func NewCardsHandler(dataAPI api.DataAPI, paymentAPI apiservice.StripeClient, addressValidationAPI address.Validator) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&cardsHandler{
			dataAPI:              dataAPI,
			paymentAPI:           paymentAPI,
			addressValidationAPI: addressValidationAPI,
		}), httputil.Get, httputil.Delete, httputil.Post, httputil.Put)
}

type PatientCardsRequestData struct {
	CardID int64 `schema:"card_id" json:"card_id,string"`
}

type PatientCardsResponse struct {
	Cards []*common.Card `json:"cards"`
}

func (p *cardsHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.RolePatient {
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (p *cardsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		p.getCardsForPatient(w, r)
	case httputil.Delete:
		p.deleteCardForPatient(w, r)
	case httputil.Put:
		p.makeCardDefaultForPatient(w, r)
	case httputil.Post:
		p.addCardForPatient(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *cardsHandler) getCardsForPatient(w http.ResponseWriter, r *http.Request) {
	patient, err := p.dataAPI.GetPatientFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, PatientCardsResponse{Cards: cards})
}

func (p *cardsHandler) makeCardDefaultForPatient(w http.ResponseWriter, r *http.Request) {
	requestData := &PatientCardsRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	card, err := p.dataAPI.GetCardFromID(requestData.CardID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patient, err := p.dataAPI.GetPatientFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	pendingTaskID, err := p.dataAPI.CreatePendingTask(api.PendingTaskPatientCard, api.StatusUpdating, patient.ID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if patient.PaymentCustomerID == "" {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.dataAPI.MakeCardDefaultForPatient(patient.ID.Int64(), card); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.paymentAPI.MakeCardDefaultForCustomer(card.ThirdPartyID, patient.PaymentCustomerID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.dataAPI.UpdateDefaultAddressForPatient(patient.ID.Int64(), card.BillingAddress); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.dataAPI.DeletePendingTask(pendingTaskID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, PatientCardsResponse{Cards: cards})
}

func (p *cardsHandler) deleteCardForPatient(w http.ResponseWriter, r *http.Request) {
	requestData := &PatientCardsRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	card, err := p.dataAPI.GetCardFromID(requestData.CardID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if card == nil {
		apiservice.WriteResourceNotFoundError("Card not found", w, r)
		return
	}

	patient, err := p.dataAPI.GetPatientFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	pendingTaskID, err := p.dataAPI.CreatePendingTask(api.PendingTaskPatientCard, api.StatusDeleting, patient.ID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if patient.PaymentCustomerID == "" {
		apiservice.WriteValidationError("Patient not registered yet for accepting payments", w, r)
		return
	}

	// mark the card as inactive instead of deleting it initially so that we have room to identify
	// situations where the call fails and things are left in an inconsistent state
	if err := p.dataAPI.MarkCardInactiveForPatient(patient.ID.Int64(), card); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	currentPatientAddressID := patient.PatientAddress.ID

	// switch over the default card to the last added card if we are currently deleting the default card
	if card.IsDefault {
		latestCard, err := p.dataAPI.MakeLatestCardDefaultForPatient(patient.ID.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if latestCard != nil {
			if err := p.dataAPI.UpdateDefaultAddressForPatient(patient.ID.Int64(), latestCard.BillingAddress); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
	}

	// the payment service changes the default card to the last added active card internally
	if err := p.paymentAPI.DeleteCardForCustomer(patient.PaymentCustomerID, card.ThirdPartyID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.dataAPI.DeleteCardForPatient(patient.ID.Int64(), card); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// delete the address only if this is not the patient's preferred address
	if currentPatientAddressID != card.BillingAddress.ID {
		if err := p.dataAPI.DeleteAddress(card.BillingAddress.ID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	if err := p.dataAPI.DeletePendingTask(pendingTaskID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, PatientCardsResponse{Cards: cards})
}

func (p *cardsHandler) addCardForPatient(w http.ResponseWriter, r *http.Request) {
	cardToAdd := &common.Card{}
	if err := json.NewDecoder(r.Body).Decode(&cardToAdd); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	//  look up the payment service customer id for the patient
	patient, err := p.dataAPI.GetPatientFromAccountID(apiservice.GetContext(r).AccountID)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError("no patient found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Make the new card the default one
	cardToAdd.IsDefault = true
	if err := addCardForPatient(p.dataAPI, p.paymentAPI, p.addressValidationAPI, cardToAdd, patient); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &PatientCardsResponse{Cards: cards})
}

func (p *cardsHandler) getCardsAndReconcileWithPaymentService(patient *common.Patient) ([]*common.Card, error) {
	localCards, err := p.dataAPI.GetCardsForPatient(patient.ID.Int64())
	if err != nil {
		return nil, err
	}

	if len(localCards) == 0 {
		return localCards, nil
	}

	stripeCards, err := p.paymentAPI.GetCardsForCustomer(patient.PaymentCustomerID)
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
	addressValidationAPI address.Validator,
	cardToAdd *common.Card,
	patient *common.Patient,
) error {
	if cardToAdd.BillingAddress == nil || cardToAdd.BillingAddress.AddressLine1 == "" || cardToAdd.BillingAddress.City == "" ||
		cardToAdd.BillingAddress.State == "" || cardToAdd.BillingAddress.ZipCode == "" {
		return apiservice.NewValidationError("Billing address for credit card not correctly specified")
	}

	if cardToAdd.Token == "" {
		return apiservice.NewValidationError("Unable to add credit card that does not have a unique token to help identify the card with the third party service")
	}

	if err := surescripts.ValidateAddress(cardToAdd.BillingAddress, addressValidationAPI, dataAPI); err != nil {
		return apiservice.NewValidationError(err.Error())
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
		if err := dataAPI.UpdatePatientWithPaymentCustomerID(patient.ID.Int64(), customer.ID); err != nil {
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
	if err := dataAPI.AddCardForPatient(patient.ID.Int64(), cardToAdd); err != nil {
		return err
	}

	// the card added for an existing patient does not become default on add; need to explicitly make a call to stripe
	// to make it the default card
	if isPatientRegisteredWithPatientService {
		if err := paymentAPI.MakeCardDefaultForCustomer(cardToAdd.ThirdPartyID, patient.PaymentCustomerID); err != nil {
			return err
		}
	}

	if err := dataAPI.UpdateDefaultAddressForPatient(patient.ID.Int64(), cardToAdd.BillingAddress); err != nil {
		return err
	}

	if err := dataAPI.DeletePendingTask(pendingTaskID); err != nil {
		return err
	}

	return nil
}

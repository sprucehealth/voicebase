package apiservice

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/payment"
)

type PatientCardsHandler struct {
	DataApi              api.DataAPI
	PaymentApi           payment.PaymentAPI
	AddressValidationApi address.AddressValidationAPI
}

type PatientCardsRequestData struct {
	CardId string `schema:"card_id"`
}

type PatientCardsResponse struct {
	Cards []*common.Card `json:"cards"`
}

func (p *PatientCardsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_GET:
		p.getCardsForPatient(w, r)
	case HTTP_DELETE:
		p.deleteCardForPatient(w, r)
	case HTTP_PUT:
		p.makeCardDefaultForPatient(w, r)
	case HTTP_POST:
		p.addCardForPatient(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *PatientCardsHandler) getCardsForPatient(w http.ResponseWriter, r *http.Request) {
	patient, err := p.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	WriteJSON(w, &PatientCardsResponse{Cards: cards})
}

func (p *PatientCardsHandler) makeCardDefaultForPatient(w http.ResponseWriter, r *http.Request) {
	requestData := &PatientCardsRequestData{}
	if err := DecodeRequestData(requestData, r); err != nil {
		WriteValidationError(err.Error(), w, r)
		return
	}

	cardId, err := strconv.ParseInt(requestData.CardId, 10, 64)
	if err != nil {
		WriteValidationError(err.Error(), w, r)
		return
	}

	card, err := p.DataApi.GetCardFromId(cardId)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	patient, err := p.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	pendingTaskId, err := p.DataApi.CreatePendingTask(api.PENDING_TASK_PATIENT_CARD, api.STATUS_UPDATING, patient.PatientId.Int64())
	if err != nil {
		WriteError(err, w, r)
		return
	}

	if patient.PaymentCustomerId == "" {
		WriteError(err, w, r)
		return
	}

	if err := p.DataApi.MakeCardDefaultForPatient(patient.PatientId.Int64(), card); err != nil {
		WriteError(err, w, r)
		return
	}

	if err := p.PaymentApi.MakeCardDefaultForCustomer(card.ThirdPartyId, patient.PaymentCustomerId); err != nil {
		WriteError(err, w, r)
		return
	}

	if err := p.DataApi.UpdateDefaultAddressForPatient(patient.PatientId.Int64(), card.BillingAddress); err != nil {
		WriteError(err, w, r)
		return
	}

	if err := p.DataApi.DeletePendingTask(pendingTaskId); err != nil {
		WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	WriteJSON(w, &PatientCardsResponse{Cards: cards})
}

func (p *PatientCardsHandler) deleteCardForPatient(w http.ResponseWriter, r *http.Request) {
	requestData := &PatientCardsRequestData{}
	if err := DecodeRequestData(requestData, r); err != nil {
		WriteValidationError(err.Error(), w, r)
		return
	}

	cardId, err := strconv.ParseInt(requestData.CardId, 10, 64)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	card, err := p.DataApi.GetCardFromId(cardId)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	if card == nil {
		WriteResourceNotFoundError("Card not found", w, r)
		return
	}

	patient, err := p.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	pendingTaskId, err := p.DataApi.CreatePendingTask(api.PENDING_TASK_PATIENT_CARD, api.STATUS_DELETING, patient.PatientId.Int64())
	if err != nil {
		WriteError(err, w, r)
		return
	}

	if patient.PaymentCustomerId == "" {
		WriteError(err, w, r)
		return
	}

	// mark the card as inactive instead of deleting it initially so that we have room to identify
	// situations where the call fails and things are left in an inconsistent state
	if err := p.DataApi.MarkCardInactiveForPatient(patient.PatientId.Int64(), card); err != nil {
		WriteError(err, w, r)
		return
	}

	currentPatientAddressId := patient.PatientAddress.Id

	// switch over the default card to the last added card if we are currently deleting the default card
	if card.IsDefault {
		latestCard, err := p.DataApi.MakeLatestCardDefaultForPatient(patient.PatientId.Int64())
		if err != nil {
			WriteError(err, w, r)
			return
		}

		if latestCard != nil {
			if err := p.DataApi.UpdateDefaultAddressForPatient(patient.PatientId.Int64(), latestCard.BillingAddress); err != nil {
				WriteError(err, w, r)
				return
			}
		}
	}

	// the payment service changes the default card to the last added active card internally
	if err := p.PaymentApi.DeleteCardForCustomer(patient.PaymentCustomerId, card.ThirdPartyId); err != nil {
		WriteError(err, w, r)
		return
	}

	if err := p.DataApi.DeleteCardForPatient(patient.PatientId.Int64(), card); err != nil {
		WriteError(err, w, r)
		return
	}

	// delete the address only if this is not the patient's preferred address
	if currentPatientAddressId != card.BillingAddress.Id {
		if err := p.DataApi.DeleteAddress(card.BillingAddress.Id); err != nil {
			WriteError(err, w, r)
			return
		}
	}

	if err := p.DataApi.DeletePendingTask(pendingTaskId); err != nil {
		WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	WriteJSON(w, &PatientCardsResponse{Cards: cards})
}

func (p *PatientCardsHandler) addCardForPatient(w http.ResponseWriter, r *http.Request) {
	cardToAdd := &common.Card{}
	if err := DecodeRequestData(cardToAdd, r); err != nil {
		WriteValidationError(err.Error(), w, r)
		return
	}

	//  look up the payment service customer id for the patient
	patient, err := p.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	if patient == nil {
		WriteResourceNotFoundError("No patient found", w, r)
		return
	}

	if cardToAdd.BillingAddress == nil || cardToAdd.BillingAddress.AddressLine1 == "" || cardToAdd.BillingAddress.City == "" ||
		cardToAdd.BillingAddress.State == "" || cardToAdd.BillingAddress.ZipCode == "" {
		WriteError(err, w, r)
		return
	}

	if cardToAdd.Token == "" {
		WriteError(err, w, r)
		return
	}

	if err := address.ValidateAddress(p.DataApi, cardToAdd.BillingAddress, p.AddressValidationApi); err != nil {
		WriteUserError(w, http.StatusBadRequest, err.Error())
		return
	}

	// create a pending task to indicate that there's work that is currently in progress
	// to add a credit card for a patient. The reason to do this is to identify any tasks that span multiple steps
	// that may fail to complete half way through and then reconcile the work through a worker
	// that cleans things up
	pendingTaskId, err := p.DataApi.CreatePendingTask(api.PENDING_TASK_PATIENT_CARD, api.STATUS_CREATING, patient.PatientId.Int64())
	if err != nil {
		WriteError(err, w, r)
		return
	}

	isPatientRegisteredWithPatientService := patient.PaymentCustomerId != ""
	var card *common.Card
	// if it does not exist, go ahead and create one with in stripe
	if !isPatientRegisteredWithPatientService {
		customer, err := p.PaymentApi.CreateCustomerWithDefaultCard(cardToAdd.Token)
		if err != nil {
			WriteError(err, w, r)
			return
		}

		// save customer id to database
		if err := p.DataApi.UpdatePatientWithPaymentCustomerId(patient.PatientId.Int64(), customer.Id); err != nil {
			WriteError(err, w, r)
			return
		}
		card = &customer.Cards[0]
		patient.PaymentCustomerId = customer.Id
	} else {
		// add another card to the customer on the payment service
		card, err = p.PaymentApi.AddCardForCustomer(cardToAdd.Token, patient.PaymentCustomerId)
		if err != nil {
			WriteError(err, w, r)
			return
		}
	}

	cardToAdd.ThirdPartyId = card.ThirdPartyId
	cardToAdd.Fingerprint = card.Fingerprint
	if err := p.DataApi.AddCardAndMakeDefaultForPatient(patient.PatientId.Int64(), cardToAdd); err != nil {
		WriteError(err, w, r)
		return
	}

	// the card added for an existing patient does not become default on add; need to explicitly make a call to stripe
	// to make it the default card
	if isPatientRegisteredWithPatientService {
		if err := p.PaymentApi.MakeCardDefaultForCustomer(cardToAdd.ThirdPartyId, patient.PaymentCustomerId); err != nil {
			WriteError(err, w, r)
			return
		}
	}

	if err := p.DataApi.UpdateDefaultAddressForPatient(patient.PatientId.Int64(), cardToAdd.BillingAddress); err != nil {
		WriteError(err, w, r)
		return
	}

	if err := p.DataApi.DeletePendingTask(pendingTaskId); err != nil {
		WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	WriteJSON(w, &PatientCardsResponse{Cards: cards})
}

func (p *PatientCardsHandler) getCardsAndReconcileWithPaymentService(patient *common.Patient) ([]*common.Card, error) {
	localCards, err := p.DataApi.GetCardsForPatient(patient.PatientId.Int64())
	if err != nil {
		return nil, err
	}

	if len(localCards) == 0 {
		return localCards, nil
	}

	cards, err := p.PaymentApi.GetCardsForCustomer(patient.PaymentCustomerId)
	if err != nil {
		return nil, err
	}

	// log this fact so that we can figure out what is going on
	if len(localCards) != len(cards) {
		golog.Warningf("Number of cards returned from payment service differs from number of cards locally stored for patient with id %d", patient.PatientId.Int64())
	}

	// trust the cards from the payment service as the source of authority
	for _, cardFromService := range cards {
		localCardFound := false
		for _, localCard := range localCards {
			if localCard.ThirdPartyId == cardFromService.ThirdPartyId {
				cardFromService.Id = localCard.Id
				cardFromService.IsDefault = localCard.IsDefault
				cardFromService.CreationDate = localCard.CreationDate
				localCardFound = true
			}
		}
		if !localCardFound {
			golog.Warningf("Local card not found in set of cards returned from payment service for patient with id %d", patient.PatientId.Int64())
		}
	}

	// sort cards by creation date so that customer seems them in the order that they entered the cards
	sort.Sort(common.ByCreationDate(cards))

	return cards, nil
}

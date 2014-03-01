package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/golog"
	"carefront/libs/payment"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type PatientCardsHandler struct {
	DataApi    api.DataAPI
	PaymentApi payment.PaymentAPI
}

type PatientCardsRequestData struct {
	CardId string `schema:"card_id"`
}

type PatientCardsResponse struct {
	Cards []common.Card `json:"cards"`
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
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from account id : "+err.Error())
		return
	}

	localCards, err := p.DataApi.GetCardsForPatient(patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get cards from db: "+err.Error())
		return
	}

	if len(localCards) == 0 {
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, &PatientCardsResponse{Cards: nil})
		return
	}

	cards, err := p.PaymentApi.GetCardsForCustomer(patient.PaymentCustomerId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get cards for patient: "+err.Error())
		return
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
				localCardFound = true
			}
		}
		if !localCardFound {
			golog.Warningf("Local card not found in set of cards returned from payment service for patient with id %d", patient.PatientId.Int64())
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &PatientCardsResponse{Cards: cards})
}

func (p *PatientCardsHandler) makeCardDefaultForPatient(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &PatientCardsRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters : "+err.Error())
		return
	}

	cardId, err := strconv.ParseInt(requestData.CardId, 10, 64)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse cardId: "+err.Error())
	}

	card, err := p.DataApi.GetCardFromId(cardId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get card from id: "+err.Error())
		return
	}

	patient, err := p.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from account id : "+err.Error())
		return
	}

	pendingTaskId, err := p.DataApi.CreatePendingTask(api.PENDING_TASK_PATIENT_CARD, api.STATUS_UPDATING, patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create pending task for adding credit card for patient: "+err.Error())
		return
	}

	if patient.PaymentCustomerId == "" {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to complete tasks because this patient is not yet registered for accepting payments: "+err.Error())
		return
	}

	if err := p.PaymentApi.MakeCardDefaultForCustomer(card.ThirdPartyId, patient.PaymentCustomerId); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to make card the default card: "+err.Error())
		return
	}

	if err := p.DataApi.MakeCardDefaultForPatient(patient.PatientId.Int64(), card); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to make card the default card on our db: "+err.Error())
		return
	}

	if err := p.DataApi.UpdateDefaultAddressForPatient(patient.PatientId.Int64(), card.BillingAddress); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update default address for patient: "+err.Error())
		return
	}

	if err := p.DataApi.DeletePendingTask(pendingTaskId); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete pending task: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

func (p *PatientCardsHandler) deleteCardForPatient(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &PatientCardsRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters : "+err.Error())
		return
	}

	cardId, err := strconv.ParseInt(requestData.CardId, 10, 64)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse cardId: "+err.Error())
		return
	}

	card, err := p.DataApi.GetCardFromId(cardId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get card from id: "+err.Error())
		return
	}

	patient, err := p.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from account id : "+err.Error())
		return
	}

	pendingTaskId, err := p.DataApi.CreatePendingTask(api.PENDING_TASK_PATIENT_CARD, api.STATUS_DELETING, patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create pending task for adding credit card for patient: "+err.Error())
		return
	}

	if patient.PaymentCustomerId == "" {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to complete tasks because this patient is not yet registered for accepting payments: "+err.Error())
		return
	}

	// mark the card as inactive instead of deleting it initially so that we have room to identify
	// situations where the call fails and things are left in an inconsistent state
	if err := p.DataApi.MarkCardInactiveForPatient(patient.PatientId.Int64(), card); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to mark card as inactive for patient: "+err.Error())
		return
	}

	// switch over the default card to the last added card if we are currently deleting the default card
	if card.IsDefault {
		latestCard, err := p.DataApi.MakeLatestCardDefaultForPatient(patient.PatientId.Int64())
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to make latest card the default card for patient: "+err.Error())
			return
		}

		if latestCard != nil {
			if err := p.DataApi.UpdateDefaultAddressForPatient(patient.PatientId.Int64(), latestCard.BillingAddress); err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update default address for patient: "+err.Error())
				return
			}
		}
	} else {
		// delete the address only if this was not the default card because there is no link to it as the patient's address
		if err := p.DataApi.DeleteAddress(card.BillingAddress.Id); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete address: "+err.Error())
			return
		}
	}

	// the payment service changes the default card to the last added active card internally
	if err := p.PaymentApi.DeleteCardForCustomer(patient.PaymentCustomerId, card.ThirdPartyId); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete card on the payment service: "+err.Error())
		return
	}

	if err := p.DataApi.DeleteCardForPatient(patient.PatientId.Int64(), card); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete card for patient: "+err.Error())
		return
	}

	if err := p.DataApi.DeletePendingTask(pendingTaskId); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete pending task: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

func (p *PatientCardsHandler) addCardForPatient(w http.ResponseWriter, r *http.Request) {
	cardToAdd := &common.Card{}
	if err := json.NewDecoder(r.Body).Decode(&cardToAdd); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	//  look up the payment service customer id for the patient
	patient, err := p.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on account id: "+err.Error())
		return
	}

	if patient == nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "No patient returned for this account id")
		return
	}

	// create a pending task to indicate that there's work that is currently in progress
	// to add a credit card for a patient. The reason to do this is to identify any tasks that span multiple steps
	// that may fail to complete half way through (due to network blip, etc), and then reconcile the work through a worker
	// that cleans things up
	pendingTaskId, err := p.DataApi.CreatePendingTask(api.PENDING_TASK_PATIENT_CARD, api.STATUS_CREATING, patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create pending task for adding credit card for patient: "+err.Error())
		return
	}

	isPatientRegisteredWithPatientService := patient.PaymentCustomerId != ""
	var card *common.Card
	// if it does not exist, go ahead and create one with in stripe
	if !isPatientRegisteredWithPatientService {
		customer, err := p.PaymentApi.CreateCustomerWithDefaultCard(cardToAdd.Token)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create customer with default card: "+err.Error())
			return
		}

		// save customer id to database
		if err := p.DataApi.UpdatePatientWithPaymentCustomerId(patient.PatientId.Int64(), customer.Id); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update patient with payment service id: "+err.Error())
			return
		}
		card = &customer.Cards[0]
	} else {
		// add another card to the customer on the payment service
		card, err = p.PaymentApi.AddCardForCustomer(cardToAdd.Token, patient.PaymentCustomerId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add card for customer: "+err.Error())
			return
		}
	}

	cardToAdd.ThirdPartyId = card.ThirdPartyId
	cardToAdd.Fingerprint = card.Fingerprint
	if err := p.DataApi.AddCardForPatient(patient.PatientId.Int64(), cardToAdd); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add new card for patient: "+err.Error())
		return
	}

	// the card added for an existing patient does not become default on add; need to explicitly make a call to stripe
	// toe make it the default card
	if isPatientRegisteredWithPatientService {
		if err := p.PaymentApi.MakeCardDefaultForCustomer(cardToAdd.ThirdPartyId, patient.PaymentCustomerId); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to make card just added the default card: "+err.Error())
			return
		}
	}

	if err := p.DataApi.UpdateDefaultAddressForPatient(patient.PatientId.Int64(), cardToAdd.BillingAddress); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update default address for patient: "+err.Error())
		return
	}

	if err := p.DataApi.DeletePendingTask(pendingTaskId); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete pending task: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, cardToAdd)
}

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
	"github.com/sprucehealth/backend/libs/payment"
)

type cardsHandler struct {
	dataAPI              api.DataAPI
	paymentAPI           payment.PaymentAPI
	addressValidationAPI address.AddressValidationAPI
}

func NewCardsHandler(dataAPI api.DataAPI, paymentAPI payment.PaymentAPI, addressValidationAPI address.AddressValidationAPI) http.Handler {
	return &cardsHandler{
		dataAPI:              dataAPI,
		paymentAPI:           paymentAPI,
		addressValidationAPI: addressValidationAPI,
	}
}

type PatientCardsRequestData struct {
	CardId int64 `schema:"card_id"`
}

type PatientCardsResponse struct {
	Cards []*common.Card `json:"cards"`
}

func (p *cardsHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (p *cardsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		p.getCardsForPatient(w, r)
	case apiservice.HTTP_DELETE:
		p.deleteCardForPatient(w, r)
	case apiservice.HTTP_PUT:
		p.makeCardDefaultForPatient(w, r)
	case apiservice.HTTP_POST:
		p.addCardForPatient(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *cardsHandler) getCardsForPatient(w http.ResponseWriter, r *http.Request) {
	patient, err := p.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, PatientCardsResponse{Cards: cards})
}

func (p *cardsHandler) makeCardDefaultForPatient(w http.ResponseWriter, r *http.Request) {
	requestData := &PatientCardsRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	card, err := p.dataAPI.GetCardFromId(requestData.CardId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patient, err := p.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	pendingTaskId, err := p.dataAPI.CreatePendingTask(api.PENDING_TASK_PATIENT_CARD, api.STATUS_UPDATING, patient.PatientId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if patient.PaymentCustomerId == "" {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.dataAPI.MakeCardDefaultForPatient(patient.PatientId.Int64(), card); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.paymentAPI.MakeCardDefaultForCustomer(card.ThirdPartyId, patient.PaymentCustomerId); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.dataAPI.UpdateDefaultAddressForPatient(patient.PatientId.Int64(), card.BillingAddress); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.dataAPI.DeletePendingTask(pendingTaskId); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, PatientCardsResponse{Cards: cards})
}

func (p *cardsHandler) deleteCardForPatient(w http.ResponseWriter, r *http.Request) {
	requestData := &PatientCardsRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	card, err := p.dataAPI.GetCardFromId(requestData.CardId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if card == nil {
		apiservice.WriteResourceNotFoundError("Card not found", w, r)
		return
	}

	patient, err := p.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	pendingTaskId, err := p.dataAPI.CreatePendingTask(api.PENDING_TASK_PATIENT_CARD, api.STATUS_DELETING, patient.PatientId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if patient.PaymentCustomerId == "" {
		apiservice.WriteValidationError("Patient not registered yet for accepting payments", w, r)
		return
	}

	// mark the card as inactive instead of deleting it initially so that we have room to identify
	// situations where the call fails and things are left in an inconsistent state
	if err := p.dataAPI.MarkCardInactiveForPatient(patient.PatientId.Int64(), card); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	currentPatientAddressId := patient.PatientAddress.Id

	// switch over the default card to the last added card if we are currently deleting the default card
	if card.IsDefault {
		latestCard, err := p.dataAPI.MakeLatestCardDefaultForPatient(patient.PatientId.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if latestCard != nil {
			if err := p.dataAPI.UpdateDefaultAddressForPatient(patient.PatientId.Int64(), latestCard.BillingAddress); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
	}

	// the payment service changes the default card to the last added active card internally
	if err := p.paymentAPI.DeleteCardForCustomer(patient.PaymentCustomerId, card.ThirdPartyId); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.dataAPI.DeleteCardForPatient(patient.PatientId.Int64(), card); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// delete the address only if this is not the patient's preferred address
	if currentPatientAddressId != card.BillingAddress.Id {
		if err := p.dataAPI.DeleteAddress(card.BillingAddress.Id); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	if err := p.dataAPI.DeletePendingTask(pendingTaskId); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, PatientCardsResponse{Cards: cards})
}

func (p *cardsHandler) addCardForPatient(w http.ResponseWriter, r *http.Request) {
	cardToAdd := &common.Card{}
	if err := json.NewDecoder(r.Body).Decode(&cardToAdd); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	//  look up the payment service customer id for the patient
	patient, err := p.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if patient == nil {
		apiservice.WriteResourceNotFoundError("no patient found", w, r)
		return
	}

	if cardToAdd.BillingAddress == nil || cardToAdd.BillingAddress.AddressLine1 == "" || cardToAdd.BillingAddress.City == "" ||
		cardToAdd.BillingAddress.State == "" || cardToAdd.BillingAddress.ZipCode == "" {
		apiservice.WriteValidationError("Billing address for credit card not correctly specified", w, r)
		return
	}

	if cardToAdd.Token == "" {
		apiservice.WriteValidationError("Unable to add credit card that does not have a unique token to help identify the card with the third party service", w, r)
		return
	}

	if err := address.ValidateAddress(p.dataAPI, cardToAdd.BillingAddress, p.addressValidationAPI); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// create a pending task to indicate that there's work that is currently in progress
	// to add a credit card for a patient. The reason to do this is to identify any tasks that span multiple steps
	// that may fail to complete half way through and then reconcile the work through a worker
	// that cleans things up
	pendingTaskId, err := p.dataAPI.CreatePendingTask(api.PENDING_TASK_PATIENT_CARD, api.STATUS_CREATING, patient.PatientId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	isPatientRegisteredWithPatientService := patient.PaymentCustomerId != ""
	var card *common.Card
	// if it does not exist, go ahead and create one with in stripe
	if !isPatientRegisteredWithPatientService {
		customer, err := p.paymentAPI.CreateCustomerWithDefaultCard(cardToAdd.Token)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// save customer id to database
		if err := p.dataAPI.UpdatePatientWithPaymentCustomerId(patient.PatientId.Int64(), customer.Id); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		card = &customer.Cards[0]
		patient.PaymentCustomerId = customer.Id
	} else {
		// add another card to the customer on the payment service
		card, err = p.paymentAPI.AddCardForCustomer(cardToAdd.Token, patient.PaymentCustomerId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	cardToAdd.ThirdPartyId = card.ThirdPartyId
	cardToAdd.Fingerprint = card.Fingerprint
	if err := p.dataAPI.AddCardAndMakeDefaultForPatient(patient.PatientId.Int64(), cardToAdd); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// the card added for an existing patient does not become default on add; need to explicitly make a call to stripe
	// to make it the default card
	if isPatientRegisteredWithPatientService {
		if err := p.paymentAPI.MakeCardDefaultForCustomer(cardToAdd.ThirdPartyId, patient.PaymentCustomerId); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	if err := p.dataAPI.UpdateDefaultAddressForPatient(patient.PatientId.Int64(), cardToAdd.BillingAddress); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := p.dataAPI.DeletePendingTask(pendingTaskId); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := p.getCardsAndReconcileWithPaymentService(patient)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &PatientCardsResponse{Cards: cards})
}

func (p *cardsHandler) getCardsAndReconcileWithPaymentService(patient *common.Patient) ([]*common.Card, error) {
	localCards, err := p.dataAPI.GetCardsForPatient(patient.PatientId.Int64())
	if err != nil {
		return nil, err
	}

	if len(localCards) == 0 {
		return localCards, nil
	}

	cards, err := p.paymentAPI.GetCardsForCustomer(patient.PaymentCustomerId)
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

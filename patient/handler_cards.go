package patient

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
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

	cards, err := getCardsAndReconcileWithPaymentService(patient, p.dataAPI, p.paymentAPI)
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

	cards, err := getCardsAndReconcileWithPaymentService(patient, p.dataAPI, p.paymentAPI)
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

	patient, err := p.dataAPI.GetPatientFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	switchDefaultCard := true
	if err := deleteCard(
		requestData.CardID,
		patient,
		switchDefaultCard,
		p.dataAPI,
		p.paymentAPI); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := getCardsAndReconcileWithPaymentService(
		patient, p.dataAPI, p.paymentAPI)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, PatientCardsResponse{
		Cards: cards,
	})
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
	enforceAddressRequirement := true
	if err := addCardForPatient(
		p.dataAPI,
		p.paymentAPI,
		p.addressValidationAPI,
		cardToAdd,
		patient,
		enforceAddressRequirement); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	cards, err := getCardsAndReconcileWithPaymentService(patient, p.dataAPI, p.paymentAPI)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &PatientCardsResponse{Cards: cards})
}

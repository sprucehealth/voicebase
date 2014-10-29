package patient

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/storage"
)

type patientVisitHandler struct {
	dataAPI              api.DataAPI
	authAPI              api.AuthAPI
	paymentAPI           apiservice.StripeClient
	addressValidationAPI address.AddressValidationAPI
	dispatcher           *dispatch.Dispatcher
	store                storage.Store
	expirationDuration   time.Duration
}

type PatientVisitRequestData struct {
	PatientVisitID int64        `schema:"patient_visit_id,required" json:"patient_visit_id,string"`
	Card           *common.Card `json:"card,omitempty"`
	ApplePay       bool         `json:"apple_pay"`
}

type PatientVisitResponse struct {
	PatientVisitId int64                         `json:"patient_visit_id,string"`
	Status         string                        `json:"status,omitempty"`
	SubmittedDate  *time.Time                    `json:"submission_date,omitempty"`
	ClientLayout   *info_intake.InfoIntakeLayout `json:"health_condition,omitempty"`
}

type PatientVisitSubmittedResponse struct {
	PatientVisitId int64  `json:"patient_visit_id,string"`
	Status         string `json:"status,omitempty"`
}

func NewPatientVisitHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, paymentAPI apiservice.StripeClient, addressValidationAPI address.AddressValidationAPI, dispatcher *dispatch.Dispatcher, store storage.Store, expirationDuration time.Duration) http.Handler {
	return &patientVisitHandler{
		dataAPI:              dataAPI,
		authAPI:              authAPI,
		paymentAPI:           paymentAPI,
		addressValidationAPI: addressValidationAPI,
		dispatcher:           dispatcher,
		store:                store,
		expirationDuration:   expirationDuration,
	}
}

func (p *patientVisitHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (s *patientVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		s.getPatientVisit(w, r)
	case apiservice.HTTP_POST:
		s.createNewPatientVisitHandler(w, r)
	case apiservice.HTTP_PUT:
		s.submitPatientVisit(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *patientVisitHandler) submitPatientVisit(w http.ResponseWriter, r *http.Request) {
	requestData := &PatientVisitRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patient, err := s.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if requestData.Card != nil {
		requestData.Card.ApplePay = requestData.ApplePay
		requestData.Card.IsDefault = false
		if err := addCardForPatient(r, s.dataAPI, s.paymentAPI, s.addressValidationAPI, requestData.Card, patient); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		// Refetch the patient object to get latest address
		patient, err = s.dataAPI.GetPatientFromId(patient.PatientId.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	visit, err := submitVisit(r, s.dataAPI, s.dispatcher, patient, requestData.PatientVisitID, 0)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &PatientVisitSubmittedResponse{
		PatientVisitId: visit.PatientVisitId.Int64(),
		Status:         visit.Status,
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}

func (s *patientVisitHandler) getPatientVisit(w http.ResponseWriter, r *http.Request) {

	patientId, err := s.dataApi.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// return the specific patient visit if ID is specified,
	// else return the last created patient visit
	var patientVisit *common.PatientVisit
	visitIdStr := r.FormValue("patient_visit_id")
	if visitIdStr != "" {
		visitID, err := strconv.ParseInt(visitIdStr, 10, 64)
		if err != nil {
			apiservice.WriteValidationError(err.Error(), w, r)
			return
		}
		patientVisit, err = s.dataApi.GetPatientVisitFromId(visitID)
		if err == api.NoRowsError {
			apiservice.WriteResourceNotFoundError("visit not found", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
		}
	} else {
		patientVisit, err = s.dataApi.GetLastCreatedPatientVisit(patientId)
		if err == api.NoRowsError {
			apiservice.WriteDeveloperErrorWithCode(w, apiservice.DEVELOPER_ERROR_NO_VISIT_EXISTS, http.StatusBadRequest, "No patient visit exists for this patient")
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	patientVisitLayout, err := GetPatientVisitLayout(s.dataAPI, s.dispatcher, s.store, s.expirationDuration, patientVisit, r)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	response := PatientVisitResponse{
		PatientVisitId: patientVisit.PatientVisitId.Int64(),
		Status:         patientVisit.Status,
		ClientLayout:   patientVisitLayout,
	}

	// add the submission date only if the visit is in a submitted state from the patient's side
	switch patientVisit.Status {
	case common.PVStatusOpen:
	default:
		response.SubmittedDate = &patientVisit.SubmittedDate
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, response)
}

func (s *patientVisitHandler) createNewPatientVisitHandler(w http.ResponseWriter, r *http.Request) {
	patient, err := s.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from the accountId retreived from the auth token: "+err.Error())
		return
	}

	pvResponse, err := createPatientVisit(patient, s.dataAPI, s.dispatcher, s.store, s.expirationDuration, r, nil)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, pvResponse)
}

func submitVisit(r *http.Request, dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, patient *common.Patient, visitID int64, cardID int64) (*common.PatientVisit, error) {
	if patient.Pharmacy == nil {
		return nil, apiservice.NewValidationError("Unable to submit the visit until a pharmacy is selected to which we can send any prescriptions", r)
	} else if patient.PatientAddress == nil {
		return nil, apiservice.NewValidationError("Unable to submit the visit until you've entered a valid credit card and billing address", r)
	}

	visit, err := dataAPI.GetPatientVisitFromId(visitID)
	if err != nil {
		return nil, apiservice.NewError(err.Error(), http.StatusBadRequest)
	}
	if visit.PatientId.Int64() != patient.PatientId.Int64() {
		return nil, apiservice.NewError("PatientID from auth token and patient id from patient visit don't match", http.StatusForbidden)
	}

	// nothing to do if the visit is already sumitted
	switch visit.Status {
	case common.PVStatusSubmitted, common.PVStatusCharged, common.PVStatusRouted:
		return visit, nil
	}

	// do not support the submitting of a case that is in another state
	if visit.Status != common.PVStatusOpen {
		return nil, apiservice.NewValidationError("Cannot submit a case that is not in the open state. Current status of case = "+visit.Status, r)
	}

	if err := dataAPI.SubmitPatientVisitWithId(visitID); err != nil {
		return nil, err
	}

	dispatcher.Publish(&VisitSubmittedEvent{
		PatientId:     patient.PatientId.Int64(),
		AccountID:     patient.AccountId.Int64(),
		VisitId:       visitID,
		PatientCaseId: visit.PatientCaseId.Int64(),
		Visit:         visit,
		CardID:        cardID,
	})

	return visit, nil
}

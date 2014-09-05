package patient

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/storage"
)

type patientVisitHandler struct {
	dataApi            api.DataAPI
	authApi            api.AuthAPI
	store              storage.Store
	expirationDuration time.Duration
}

type patientVisitRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id,required"`
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

func NewPatientVisitHandler(dataApi api.DataAPI, authApi api.AuthAPI, store storage.Store, expirationDuration time.Duration) http.Handler {
	return &patientVisitHandler{
		dataApi:            dataApi,
		authApi:            authApi,
		store:              store,
		expirationDuration: expirationDuration,
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
		s.returnLastCreatedPatientVisit(w, r)
	case apiservice.HTTP_POST:
		s.createNewPatientVisitHandler(w, r)
	case apiservice.HTTP_PUT:
		s.submitPatientVisit(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *patientVisitHandler) submitPatientVisit(w http.ResponseWriter, r *http.Request) {
	requestData := &patientVisitRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patient, err := s.dataApi.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	} else if patient.Pharmacy == nil {
		apiservice.WriteValidationError("Unable to submit the visit until a pharmacy is selected to which we can send any prescriptions", w, r)
		return
	} else if patient.PatientAddress == nil {
		apiservice.WriteValidationError("Unable to submit the visit until you've entered a valid credit card and billing address", w, r)
		return
	}

	patientIdFromPatientVisitId, err := s.dataApi.GetPatientIdFromPatientVisitId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if patient.PatientId.Int64() != patientIdFromPatientVisitId {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "PatientId from auth token and patient id from patient visit don't match")
		return
	}

	patientVisit, err := s.dataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	// nothing to do if the visit is already sumitted
	switch patientVisit.Status {
	case common.PVStatusSubmitted, common.PVStatusCharged, common.PVStatusRouted:
		return
	}

	// do not support the submitting of a case that is in another state
	if patientVisit.Status != common.PVStatusOpen {
		apiservice.WriteValidationError("Cannot submit a case that is not in the open state. Current status of case = "+patientVisit.Status, w, r)
		return
	}

	err = s.dataApi.SubmitPatientVisitWithId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dispatch.Default.Publish(&VisitSubmittedEvent{
		PatientId:     patient.PatientId.Int64(),
		VisitId:       requestData.PatientVisitId,
		PatientCaseId: patientVisit.PatientCaseId.Int64(),
		Visit:         patientVisit,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientVisitSubmittedResponse{PatientVisitId: patientVisit.PatientVisitId.Int64(), Status: patientVisit.Status})
}

func (s *patientVisitHandler) returnLastCreatedPatientVisit(w http.ResponseWriter, r *http.Request) {

	patientId, err := s.dataApi.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// get the last created patient visit for this patient
	patientVisit, err := s.dataApi.GetLastCreatedPatientVisit(patientId)
	if err != nil {
		if err == api.NoRowsError {
			apiservice.WriteDeveloperErrorWithCode(w, apiservice.DEVELOPER_ERROR_NO_VISIT_EXISTS, http.StatusBadRequest, "No patient visit exists for this patient")
			return
		}

		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	patientVisitLayout, err := GetPatientVisitLayout(s.dataApi, s.store, s.expirationDuration, patientVisit, r)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
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

func GetPatientVisitLayout(dataApi api.DataAPI, store storage.Store, expirationDuration time.Duration, patientVisit *common.PatientVisit, r *http.Request) (*info_intake.InfoIntakeLayout, error) {

	// if there is an active patient visit record, then ensure to lookup the layout to send to the patient
	// based on what layout was shown to the patient at the time of opening of the patient visit, NOT the current
	// based on what is the current active layout because that may have potentially changed and we want to ensure
	// to not confuse the patient by changing the question structure under their feet for this particular patient visit
	// in other words, want to show them what they have already seen in terms of a flow.
	patientVisitLayout, err := apiservice.GetPatientLayoutForPatientVisit(patientVisit, api.EN_LANGUAGE_ID, dataApi)
	if err != nil {
		return nil, err
	}

	err = populateGlobalSectionsWithPatientAnswers(dataApi, store, expirationDuration, patientVisitLayout, patientVisit.PatientId.Int64(), r)
	if err != nil {
		return nil, err
	}

	err = populateSectionsWithPatientAnswers(dataApi, store, expirationDuration, patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64(), patientVisitLayout, r)
	if err != nil {
		return nil, err
	}
	return patientVisitLayout, nil
}

func (s *patientVisitHandler) createNewPatientVisitHandler(w http.ResponseWriter, r *http.Request) {
	patient, err := s.dataApi.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from the accountId retreived from the auth token: "+err.Error())
		return
	}

	pvResponse, err := createPatientVisit(patient, s.dataApi, s.store, s.expirationDuration, r)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, pvResponse)
}

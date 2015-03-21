package patient

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
)

type patientVisitHandler struct {
	dataAPI              api.DataAPI
	authAPI              api.AuthAPI
	paymentAPI           apiservice.StripeClient
	addressValidationAPI address.AddressValidationAPI
	apiDomain            string
	dispatcher           *dispatch.Dispatcher
	mediaStore           *media.Store
	expirationDuration   time.Duration
}

type PatientVisitRequestData struct {
	PatientVisitID int64        `schema:"patient_visit_id" json:"patient_visit_id,string"`
	PathwayTag     string       `schema:"pathway_id" json:"pathway_id"`
	DoctorID       int64        `schema:"care_provider_id" json:"care_provider_id,string"`
	Card           *common.Card `json:"card,omitempty"`
	ApplePay       bool         `json:"apple_pay"`
}

type PatientVisitResponse struct {
	*VisitIntakeInfo
	SubmittedDate *time.Time `json:"submission_date,omitempty"`
}

type VisitIntakeInfo struct {
	PatientVisitID int64                         `json:"patient_visit_id,string"`
	DoctorID       int64                         `json:"care_provider_id,string,omitempty"`
	CanAbandon     bool                          `json:"can_abandon"`
	Status         string                        `json:"status,omitempty"`
	ClientLayout   *info_intake.InfoIntakeLayout `json:"health_condition,omitempty"`
}

type PatientVisitSubmittedResponse struct {
	PatientVisitID int64  `json:"patient_visit_id,string"`
	Status         string `json:"status,omitempty"`
}

func NewPatientVisitHandler(
	dataAPI api.DataAPI,
	authAPI api.AuthAPI,
	paymentAPI apiservice.StripeClient,
	addressValidationAPI address.AddressValidationAPI,
	apiDomain string,
	dispatcher *dispatch.Dispatcher,
	mediaStore *media.Store,
	expirationDuration time.Duration,
) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&patientVisitHandler{
					dataAPI:              dataAPI,
					authAPI:              authAPI,
					paymentAPI:           paymentAPI,
					addressValidationAPI: addressValidationAPI,
					apiDomain:            apiDomain,
					dispatcher:           dispatcher,
					mediaStore:           mediaStore,
					expirationDuration:   expirationDuration,
				}), []string{api.PATIENT_ROLE}), []string{httputil.Get, httputil.Post, httputil.Put, httputil.Delete})
}

func (s *patientVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		s.getPatientVisit(w, r)
	case httputil.Post:
		s.createNewPatientVisitHandler(w, r)
	case httputil.Put:
		s.submitPatientVisit(w, r)
	case httputil.Delete:
		s.deletePatientVisit(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *patientVisitHandler) deletePatientVisit(w http.ResponseWriter, r *http.Request) {
	requestData := &PatientVisitRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	} else if requestData.PatientVisitID == 0 {
		apiservice.WriteValidationError("patient_visit_id required", w, r)
		return
	}

	visit, err := s.dataAPI.GetPatientVisitFromID(requestData.PatientVisitID)
	if err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// only allowed to abandon the initial visit to a case for now
	if visit.IsFollowup {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	} else if visit.Status != common.PVStatusOpen && visit.Status != common.PVStatusDeleted {
		// can only delete an open visit
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	// update the visit to mark it as deleted
	visitStatus := common.PVStatusDeleted
	if err := s.dataAPI.UpdatePatientVisit(visit.PatientVisitID.Int64(), &api.PatientVisitUpdate{
		Status: &visitStatus,
	}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// update the case to mark it as deleted
	caseStatus := common.PCStatusDeleted
	if err := s.dataAPI.UpdatePatientCase(visit.PatientCaseID.Int64(), &api.PatientCaseUpdate{
		Status: &caseStatus,
	}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

func (s *patientVisitHandler) submitPatientVisit(w http.ResponseWriter, r *http.Request) {
	requestData := &PatientVisitRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	} else if requestData.PatientVisitID == 0 {
		apiservice.WriteValidationError("missing patient_visit_id", w, r)
		return
	}

	patient, err := s.dataAPI.GetPatientFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var cardID int64
	if requestData.Card != nil {
		requestData.Card.ApplePay = requestData.ApplePay
		requestData.Card.IsDefault = true
		if err := addCardForPatient(s.dataAPI, s.paymentAPI, s.addressValidationAPI, requestData.Card, patient); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		// Refetch the patient object to get latest address
		patient, err = s.dataAPI.GetPatientFromID(patient.PatientID.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		cardID = requestData.Card.ID.Int64()
	}

	visit, err := submitVisit(s.dataAPI, s.dispatcher, patient, requestData.PatientVisitID, cardID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &PatientVisitSubmittedResponse{
		PatientVisitID: visit.PatientVisitID.Int64(),
		Status:         visit.Status,
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

func (s *patientVisitHandler) getPatientVisit(w http.ResponseWriter, r *http.Request) {

	patientID, err := s.dataAPI.GetPatientIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// return the specific patient visit if ID is specified,
	// else return the last created patient visit
	var patientVisit *common.PatientVisit
	visitIDStr := r.FormValue("patient_visit_id")
	if visitIDStr != "" {
		visitID, err := strconv.ParseInt(visitIDStr, 10, 64)
		if err != nil {
			apiservice.WriteValidationError(err.Error(), w, r)
			return
		}
		patientVisit, err = s.dataAPI.GetPatientVisitFromID(visitID)
		if api.IsErrNotFound(err) {
			apiservice.WriteResourceNotFoundError("visit not found", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	} else {

		// return the last created patient visit for the active case for the assumed ACNE pathway.
		// NOTE: the call to get a visit without a patient_visit_id only exists for backwards compatibility
		// reasons where v1.0 of the iOS client assumed a single visit existed for the patient
		// and so did not pass in a patient_visit_id parameter
		patientCases, err := s.dataAPI.CasesForPathway(patientID, api.AcnePathwayTag, []string{common.PCStatusActive.String(), common.PCStatusOpen.String()})
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		if len(patientCases) > 1 {
			apiservice.WriteError(fmt.Errorf("Expected single active case for pathway %s but got %d", api.AcnePathwayTag, len(patientCases)), w, r)
			return
		} else if len(patientCases) == 0 {
			apiservice.WriteResourceNotFoundError(fmt.Sprintf("no active case exists for pathway %s", api.AcnePathwayTag), w, r)
			return
		}

		patientVisits, err := s.dataAPI.GetVisitsForCase(patientCases[0].ID.Int64(), common.OpenPatientVisitStates())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		} else if len(patientVisits) == 0 {
			apiservice.WriteResourceNotFoundError("no patient visit exists", w, r)
			return
		}

		// return the latest open patient visit for the case
		sort.Reverse(common.ByPatientVisitCreationDate(patientVisits))
		patientVisit = patientVisits[0]
	}

	if patientVisit.Status == common.PVStatusPending {
		if err := checkLayoutVersionForFollowup(s.dataAPI, s.dispatcher, patientVisit, r); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	intakeInfo, err := IntakeLayoutForVisit(s.dataAPI, s.apiDomain, s.mediaStore, s.expirationDuration, patientVisit)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	response := PatientVisitResponse{
		VisitIntakeInfo: intakeInfo,
	}

	// add the submission date only if the visit is in a submitted state from the patient's side
	switch patientVisit.Status {
	case common.PVStatusOpen:
	default:
		response.SubmittedDate = &patientVisit.SubmittedDate
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}

func (s *patientVisitHandler) createNewPatientVisitHandler(w http.ResponseWriter, r *http.Request) {
	var rq PatientVisitRequestData
	if err := apiservice.DecodeRequestData(&rq, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patient, err := s.dataAPI.GetPatientFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from the accountId retreived from the auth token: "+err.Error())
		return
	}
	if rq.PathwayTag == "" {
		// assume acne for backwards compatibility
		rq.PathwayTag = api.AcnePathwayTag
	}

	pvResponse, err := createPatientVisit(
		patient,
		rq.DoctorID,
		rq.PathwayTag,
		s.dataAPI,
		s.apiDomain,
		s.dispatcher,
		s.mediaStore,
		s.expirationDuration, r, nil)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, pvResponse)
}

func submitVisit(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, patient *common.Patient, visitID int64, cardID int64) (*common.PatientVisit, error) {
	if patient.Pharmacy == nil {
		return nil, apiservice.NewValidationError("Unable to submit the visit until a pharmacy is selected to which we can send any prescriptions")
	} else if patient.PatientAddress == nil {
		return nil, apiservice.NewValidationError("Unable to submit the visit until you've entered a valid credit card and billing address")
	}

	visit, err := dataAPI.GetPatientVisitFromID(visitID)
	if err != nil {
		return nil, apiservice.NewError(err.Error(), http.StatusBadRequest)
	}
	if visit.PatientID.Int64() != patient.PatientID.Int64() {
		return nil, apiservice.NewError("PatientID from auth token and patient id from patient visit don't match", http.StatusForbidden)
	}

	// nothing to do if the visit is already sumitted
	switch visit.Status {
	case common.PVStatusSubmitted, common.PVStatusCharged, common.PVStatusRouted:
		return visit, nil
	}

	// do not support the submitting of a case that is in another state
	if visit.Status != common.PVStatusOpen {
		return nil, apiservice.NewValidationError("Cannot submit a case that is not in the open state. Current status of case = " + visit.Status)
	}

	if err := dataAPI.SubmitPatientVisitWithID(visitID); err != nil {
		return nil, err
	}

	dispatcher.Publish(&VisitSubmittedEvent{
		PatientID:     patient.PatientID.Int64(),
		AccountID:     patient.AccountID.Int64(),
		VisitID:       visitID,
		PatientCaseID: visit.PatientCaseID.Int64(),
		Visit:         visit,
		CardID:        cardID,
	})

	return visit, nil
}

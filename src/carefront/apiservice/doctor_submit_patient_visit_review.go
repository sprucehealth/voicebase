package apiservice

import (
	"carefront/api"
	"github.com/gorilla/schema"
	"net/http"
)

type DoctorSubmitPatientVisitReviewHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type SubmitPatientVisitReviewRequest struct {
	PatientVisitId int64 `schema:"patient_visit_id"`
}

type SubmitPatientVisitReviewResponse struct {
	Result string `json:"result"`
}

func (d *DoctorSubmitPatientVisitReviewHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func (d *DoctorSubmitPatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		d.submitPatientVisitReview(w, r)
	default:
		WriteJSONToHTTPResponseWriter(w, http.StatusNotFound, nil)
	}
}

func (d *DoctorSubmitPatientVisitReviewHandler) submitPatientVisitReview(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(SubmitPatientVisitReviewRequest)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, d.accountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	// doctor can only update the state of a patient visit that is currently in REVIEWING state
	patientVisit, err := d.DataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient visit for the patient visit id specified: "+err.Error())
		return
	}

	if patientVisit.Status != api.CASE_STATUS_REVIEWING {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to change the state of a patient visit to CLOSED when its not in the reviewing state")
		return
	}

	// update the status of the patient visit
	err = d.DataApi.ClosePatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the visit to closed: "+err.Error())
		return
	}

	// update the item in the doctors queue to say completed
	err = d.DataApi.UpdateStateForPatientVisitInDoctorQueue(doctorId, requestData.PatientVisitId, api.QUEUE_ITEM_STATUS_ONGOING, api.QUEUE_ITEM_STATUS_COMPLETED)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to mark the patient visit in the doctor's queue as completed: "+err.Error())
		return
	}

	// TODO Queue up notification to patient

	// TODO Send prescriptions to pharmacy of patient's choice

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &SubmitPatientVisitReviewResponse{Result: "success"})
}

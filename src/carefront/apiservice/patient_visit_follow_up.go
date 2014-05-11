package apiservice

import (
	"carefront/api"
	"carefront/common"
	"net/http"
	"time"

	"github.com/gorilla/schema"
)

type PatientVisitFollowUpHandler struct {
	DataApi api.DataAPI
}

type PatientVisitFollowUpRequestResponse struct {
	PatientVisitId  int64  `schema:"patient_visit_id"`
	TreatmentPlanId int64  `schema:"treatment_plan_id,omitempty"`
	FollowUpValue   int64  `schema:"follow_up_value"`
	FollowUpUnit    string `schema:"follow_up_unit"`
}

type PatientVisitFollowupResponse struct {
	Result string `json:"result,omitempty"`
	*common.FollowUp
}

func NewPatientVisitFollowUpHandler(dataApi api.DataAPI) *PatientVisitFollowUpHandler {
	return &PatientVisitFollowUpHandler{DataApi: dataApi}
}

func (p *PatientVisitFollowUpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_POST:
		p.updatePatientVisitFollowup(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *PatientVisitFollowUpHandler) updatePatientVisitFollowup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse form data: "+err.Error())
		return
	}

	requestData := new(PatientVisitFollowUpRequestResponse)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	err = EnsurePatientVisitInExpectedStatus(p.DataApi, requestData.PatientVisitId, api.CASE_STATUS_REVIEWING)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	switch requestData.FollowUpUnit {
	case api.FOLLOW_UP_WEEK, api.FOLLOW_UP_DAY, api.FOLLOW_UP_MONTH:
	default:
		WriteDeveloperError(w, http.StatusBadRequest, "Follow up unit should be week, month or day")
		return
	}

	patientVisitReviewData, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, GetContext(r).AccountId, p.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	treatmentPlanId, err := p.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for patient visit: "+err.Error())
		return
	}

	err = p.DataApi.UpdateFollowUpTimeForPatientVisit(treatmentPlanId, time.Now().Unix(), patientVisitReviewData.DoctorId, requestData.FollowUpValue, requestData.FollowUpUnit)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update followup for patient visit")
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &PatientVisitFollowupResponse{Result: "success"})
}

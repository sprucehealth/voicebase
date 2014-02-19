package apiservice

import (
	"carefront/api"
	"carefront/common"
	"net/http"

	"github.com/gorilla/schema"
)

type DiagnosisSummaryHandler struct {
	DataApi api.DataAPI
}

type DiagnosisSummaryRequestData struct {
	PatientVisitId  int64 `schema:"patient_visit_id"`
	TreatmentPlanId int64 `schema:"treatment_plan_id"`
}

func (d *DiagnosisSummaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getDiagnosisSummaryForPatientVisit(w, r)
	case "POST":
		WriteJSONToHTTPResponseWriter(w, http.StatusNotFound, nil)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (d *DiagnosisSummaryHandler) getDiagnosisSummaryForPatientVisit(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	var requestData DiagnosisSummaryRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	if err := ensureTreatmentPlanOrPatientVisitIdPresent(d.DataApi, treatmentPlanId, &patientVisitId); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisitReviewData, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	if treatmentPlanId == 0 {
		treatmentPlanId, err = d.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, requestData.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to get treatment plan for patient visit: "+err.Error())
			return
		}
	}

	summary, err := d.DataApi.GetDiagnosisSummaryForPatientVisit(patientVisitId, treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get diagnosis summary for patient visit: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &common.DiagnosisSummary{Type: "text", Summary: summary})
}

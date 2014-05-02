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
	PatientVisitId  int64  `schema:"patient_visit_id"`
	TreatmentPlanId int64  `schema:"treatment_plan_id"`
	Summary         string `schema:"summary"`
}

func (d *DiagnosisSummaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case HTTP_GET, HTTP_PUT:
	default:
		w.WriteHeader(http.StatusNotFound)
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

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
		treatmentPlanId, err = d.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, patientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get active treatment plan id based on patient visit id : "+err.Error())
			return
		}
	}

	if r.Method == HTTP_GET {
		diagnosisSummary, err := d.DataApi.GetDIagnosisForTreatmentPlan(treatmentPlanId)
		if err != nil && err != api.NoRowsError {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get diagnosis summary for patient visit: "+err.Error())
			return
		}
		var summary string
		if diagnosisSummary != nil {
			summary = diagnosisSummary.Summary
		}
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, &common.DiagnosisSummary{Type: "text", Summary: summary})
	} else if r.Method == HTTP_PUT {
		if requestData.Summary == "" {
			WriteDeveloperError(w, http.StatusBadRequest, "Summary to patient cannot be empty")
			return
		}

		if err := d.DataApi.AddOrUpdateDiagnosisSummaryForTreatmentPlan(requestData.Summary, treatmentPlanId, patientVisitReviewData.DoctorId, true); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update doctor diagnosis summary: "+err.Error())
			return
		}
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
	}
}

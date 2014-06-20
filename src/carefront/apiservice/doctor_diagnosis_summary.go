package apiservice

import (
	"carefront/api"
	"carefront/common"
	"net/http"
)

type DiagnosisSummaryHandler struct {
	DataApi api.DataAPI
}

type DiagnosisSummaryRequestData struct {
	TreatmentPlanId int64  `schema:"treatment_plan_id"`
	Summary         string `schema:"summary"`
}

func (d *DiagnosisSummaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case HTTP_GET, HTTP_PUT:
	default:
		w.WriteHeader(http.StatusNotFound)
	}

	var requestData DiagnosisSummaryRequestData
	if err := DecodeRequestData(&requestData, r); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	} else if requestData.TreatmentPlanId == 0 {
		WriteValidationError("treatment_plan_id needs to be specified", w, r)
		return
	}

	patientVisitId, err := d.DataApi.GetPatientVisitIdFromTreatmentPlanId(requestData.TreatmentPlanId)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	patientVisitReviewData, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	if r.Method == HTTP_GET {
		diagnosisSummary, err := d.DataApi.GetDiagnosisSummaryForTreatmentPlan(requestData.TreatmentPlanId)
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

		if requestData.TreatmentPlanId == 0 {
			WriteDeveloperError(w, http.StatusNotFound, "Unable to update diagnosis summary because treatment plan doesn't exist yet")
			return
		}

		if requestData.Summary == "" {
			WriteDeveloperError(w, http.StatusBadRequest, "Summary to patient cannot be empty")
			return
		}

		if err := d.DataApi.AddOrUpdateDiagnosisSummaryForTreatmentPlan(requestData.Summary, requestData.TreatmentPlanId, patientVisitReviewData.DoctorId, true); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update doctor diagnosis summary: "+err.Error())
			return
		}
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
	}
}

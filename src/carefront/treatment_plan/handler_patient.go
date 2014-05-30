package treatment_plan

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"net/http"

	"github.com/gorilla/schema"
)

type patientTreatmentPlanHandler struct {
	dataApi api.DataAPI
}

func NewPatientTreatmentPlanHandler(dataApi api.DataAPI) *patientTreatmentPlanHandler {
	return &patientTreatmentPlanHandler{
		dataApi: dataApi,
	}
}

type PatientVisitReviewRequest struct {
	PatientVisitId int64 `schema:"patient_visit_id"`
}

func (p *patientTreatmentPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData PatientVisitReviewRequest
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patient, err := p.dataApi.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from accountId retrieved from auth token: "+err.Error())
		return
	}

	var patientVisit *common.PatientVisit

	if requestData.PatientVisitId != 0 {
		patientIdFromPatientVisitId, err := p.dataApi.GetPatientIdFromPatientVisitId(requestData.PatientVisitId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from patientVisitId: "+err.Error())
			return
		}

		if patient.PatientId.Int64() != patientIdFromPatientVisitId {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "PatientId from auth token and patient id from patient visit don't match")
			return
		}

		patientVisit, err = p.dataApi.GetPatientVisitFromId(requestData.PatientVisitId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient visit from id: "+err.Error())
			return
		}
	} else {
		patientVisit, err = p.dataApi.GetLatestClosedPatientVisitForPatient(patient.PatientId.Int64())
		if err != nil {
			if err == api.NoRowsError {
				// no patient visit review to return
				apiservice.WriteDeveloperErrorWithCode(w, apiservice.DEVELOPER_NO_TREATMENT_PLAN, http.StatusNotFound, "No treatment plan exists for this patient visit yet")
				return
			}

			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get latest closed patient visit from id: "+err.Error())
			return
		}
	}

	// do not support the submitting of a case that has already been submitted or is in another state
	if patientVisit.Status != api.CASE_STATUS_TREATED && patientVisit.Status != api.CASE_STATUS_CLOSED {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Cannot get the review for a case that is not in the closed state "+patientVisit.Status)
		return
	}

	doctor, err := p.dataApi.GetDoctorAssignedToPatientVisit(patientVisit.PatientVisitId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor assigned to patient visit: "+err.Error())
		return
	}

	treatmentPlanId, err := p.dataApi.GetActiveTreatmentPlanForPatientVisit(doctor.DoctorId.Int64(), patientVisit.PatientVisitId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan based on patient visit: "+err.Error())
		return
	}

	treatmentPlan, err := populateTreatmentPlan(p.dataApi, patientVisit.PatientVisitId.Int64(), treatmentPlanId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	treatmentPlanResponse(p.dataApi, w, r, treatmentPlan, patientVisit, doctor, patient)
}

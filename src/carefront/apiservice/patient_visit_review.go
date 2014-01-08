package apiservice

import (
	"carefront/api"
	"carefront/common"
	"fmt"
	"github.com/gorilla/schema"
	"net/http"
	"strings"
)

type PatientVisitReviewHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type PatientVisitReviewRequest struct {
	PatientVisitId int64 `schema:"patient_visit_id"`
}

type PatientVisitReviewResponse struct {
	DiagnosisSummary *common.DiagnosisSummary `json:"diagnosis_summary,omitempty"`
	TreatmentPlan    *common.TreatmentPlan    `json:"treatment_plan,omitempty"`
	RegimenPlan      *common.RegimenPlan      `json:"regimen_plan,omitempty"`
	Advice           *common.Advice           `json:"advice,omitempty"`
	Followup         *common.FollowUp         `json:"follow_up,omitempty"`
}

func (p *PatientVisitReviewHandler) AccountIdFromAuthToken(accountId int64) {
	p.accountId = accountId
}

func (p *PatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(PatientVisitReviewRequest)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientId, err := p.DataApi.GetPatientIdFromAccountId(p.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from accountId retrieved from auth token: "+err.Error())
		return
	}

	patientIdFromPatientVisitId, err := p.DataApi.GetPatientIdFromPatientVisitId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from patientVisitId: "+err.Error())
		return
	}

	if patientId != patientIdFromPatientVisitId {
		WriteDeveloperError(w, http.StatusBadRequest, "PatientId from auth token and patient id from patient visit don't match")
		return
	}

	patientVisit, err := p.DataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient visit from id: "+err.Error())
		return
	}

	// do not support the submitting of a case that has already been submitted or is in another state
	if patientVisit.Status != api.CASE_STATUS_CLOSED {
		WriteDeveloperError(w, http.StatusBadRequest, "Cannot get the review for a case that is not in the closed state "+patientVisit.Status)
		return
	}

	doctor, err := p.DataApi.GetDoctorAssignedToPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor assigned to patient visit: "+err.Error())
		return
	}

	patientVisitReviewResponse := &PatientVisitReviewResponse{}

	summary, err := p.DataApi.GetDiagnosisSummaryForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get diagnosis summary for patient visit: "+err.Error())
		return
	}

	if summary != "" {
		diagnosisSummary := &common.DiagnosisSummary{}
		diagnosisSummary.Type = "text"
		diagnosisSummary.Summary = summary
		diagnosisSummary.Title = fmt.Sprintf("Message from Dr. %s", strings.Title(doctor.LastName))
		patientVisitReviewResponse.DiagnosisSummary = diagnosisSummary
	}

	treatmentPlan, err := p.DataApi.GetTreatmentPlanForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for this patient visit id: "+err.Error())
		return
	}

	treatmentPlan.Title = "Treatments"
	patientVisitReviewResponse.TreatmentPlan = treatmentPlan

	regimenPlan, err := p.DataApi.GetRegimenPlanForPatientVisit(requestData.PatientVisitId)
	if err != nil && err != api.NoRegimenPlanForPatientVisit {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get regimen plan for this patient visit id: "+err.Error())
		return
	}
	if regimenPlan != nil {
		regimenPlan.Title = "Personal Regimen"
		patientVisitReviewResponse.RegimenPlan = regimenPlan
	}

	followUp, err := p.DataApi.GetFollowUpTimeForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get follow up information for this paitent visit: "+err.Error())
		return
	}
	if followUp != nil {
		followUp.Title = "Follow Up"
		patientVisitReviewResponse.Followup = followUp
	}

	advicePoints, err := p.DataApi.GetAdvicePointsForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice for patient visit: "+err.Error())
		return
	}

	if advicePoints != nil && len(advicePoints) > 0 {
		advice := &common.Advice{}
		advice.SelectedAdvicePoints = advicePoints
		advice.Title = fmt.Sprintf("Dr. %s's Advice", strings.Title(doctor.LastName))
		patientVisitReviewResponse.Advice = advice
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, patientVisitReviewResponse)
}

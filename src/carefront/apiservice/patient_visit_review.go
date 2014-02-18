package apiservice

import (
	"carefront/api"
	"carefront/common"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/schema"
)

type PatientVisitReviewHandler struct {
	DataApi api.DataAPI
}

type PatientVisitReviewRequest struct {
	PatientVisitId int64 `schema:"patient_visit_id"`
}

type treatmentDisplayItem struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OTC         bool   `json:"otc"`
}

type treatmentsDisplaySection struct {
	Medications []*treatmentDisplayItem `json:"medications"`
	Title       string                  `json:"title"`
}

type PatientVisitReviewResponse struct {
	PatientVisitId   int64                     `json:"patient_visit_id,string,omitempty"`
	DiagnosisSummary *common.DiagnosisSummary  `json:"diagnosis_summary,omitempty"`
	Treatments       *treatmentsDisplaySection `json:"treatments,omitempty"`
	RegimenPlan      *common.RegimenPlan       `json:"regimen,omitempty"`
	Advice           *common.Advice            `json:"advice,omitempty"`
	Followup         *common.FollowUp          `json:"follow_up,omitempty"`
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

	patientId, err := p.DataApi.GetPatientIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from accountId retrieved from auth token: "+err.Error())
		return
	}

	var patientVisit *common.PatientVisit

	if requestData.PatientVisitId != 0 {
		patientIdFromPatientVisitId, err := p.DataApi.GetPatientIdFromPatientVisitId(requestData.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from patientVisitId: "+err.Error())
			return
		}

		if patientId != patientIdFromPatientVisitId {
			WriteDeveloperError(w, http.StatusBadRequest, "PatientId from auth token and patient id from patient visit don't match")
			return
		}

		patientVisit, err = p.DataApi.GetPatientVisitFromId(requestData.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient visit from id: "+err.Error())
			return
		}
	} else {
		patientVisit, err = p.DataApi.GetLatestClosedPatientVisitForPatient(patientId)
		if err != nil {

			if err == api.NoRowsError {
				// no patient visit review to return
				WriteJSONToHTTPResponseWriter(w, http.StatusOK, &common.TreatmentPlan{})
				return
			}

			WriteDeveloperError(w, http.StatusBadRequest, "Unable to get latest closed patient visit from id: "+err.Error())
			return
		}
	}

	// do not support the submitting of a case that has already been submitted or is in another state
	if patientVisit.Status != api.CASE_STATUS_TREATED && patientVisit.Status != api.CASE_STATUS_CLOSED {
		WriteDeveloperError(w, http.StatusBadRequest, "Cannot get the review for a case that is not in the closed state "+patientVisit.Status)
		return
	}

	doctor, err := p.DataApi.GetDoctorAssignedToPatientVisit(patientVisit.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor assigned to patient visit: "+err.Error())
		return
	}

	treatmentPlanId, err := p.DataApi.GetActiveTreatmentPlanForPatientVisit(doctor.DoctorId, patientVisit.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan based on patient visit: "+err.Error())
		return
	}

	patientVisitReviewResponse := &PatientVisitReviewResponse{}
	patientVisitReviewResponse.PatientVisitId = patientVisit.PatientVisitId

	summary, err := p.DataApi.GetDiagnosisSummaryForPatientVisit(patientVisit.PatientVisitId, treatmentPlanId)
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

	treatments, err := p.DataApi.GetTreatmentsBasedOnTreatmentPlanId(patientVisit.PatientVisitId, treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for this patient visit id: "+err.Error())
		return
	}

	if treatments != nil {
		treatmentsSection := &treatmentsDisplaySection{}
		treatmentsSection.Title = "Treatments"
		treatmentsSection.Medications = make([]*treatmentDisplayItem, 0)
		for _, treatment := range treatments {
			drugName, _, _ := breakDrugInternalNameIntoComponents(treatment.DrugInternalName)
			treatmentItem := &treatmentDisplayItem{}
			treatmentItem.Name = fmt.Sprintf("%s %s", drugName, treatment.DosageStrength)
			treatmentItem.Description = treatment.PatientInstructions
			treatmentItem.OTC = treatment.OTC
			treatmentsSection.Medications = append(treatmentsSection.Medications, treatmentItem)
		}

		patientVisitReviewResponse.Treatments = treatmentsSection
	}

	regimenPlan, err := p.DataApi.GetRegimenPlanForPatientVisit(patientVisit.PatientVisitId, treatmentPlanId)
	if err != nil && err != api.NoRegimenPlanForPatientVisit {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get regimen plan for this patient visit id: "+err.Error())
		return
	}
	if regimenPlan != nil {
		regimenPlan.Title = "Personal Regimen"
		patientVisitReviewResponse.RegimenPlan = regimenPlan
	}

	followUp, err := p.DataApi.GetFollowUpTimeForPatientVisit(patientVisit.PatientVisitId, treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get follow up information for this paitent visit: "+err.Error())
		return
	}
	if followUp != nil {
		followUp.Title = "Follow Up"
		patientVisitReviewResponse.Followup = followUp
	}

	advicePoints, err := p.DataApi.GetAdvicePointsForPatientVisit(patientVisit.PatientVisitId, treatmentPlanId)
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

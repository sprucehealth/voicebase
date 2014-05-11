package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type DoctorTreatmentPlanHandler struct {
	DataApi api.DataAPI
}

type DoctorTreatmentPlanRequestData struct {
	DoctorFavoriteTreatmentPlanId string `schema:"dr_favorite_treatment_plan_id"`
	PatientVisitId                string `schema:"patient_visit_id,required"`
	Abbreviated                   bool   `schema:"abbreviated"`
}

type DoctorTreatmentPlanResponse struct {
	TreatmentPlan *common.DoctorTreatmentPlan `json:"treatment_plan"`
}

func (d *DoctorTreatmentPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_GET:
		d.getTreatmentPlanForPatientVisit(w, r)
	case HTTP_PUT:
		d.pickATreatmentPlan(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func getPatientVisitIdFromRequest(r *http.Request) (int64, *DoctorTreatmentPlanRequestData, error) {
	if err := r.ParseForm(); err != nil {
		return 0, nil, errors.New("Unable to parse input parameters: " + err.Error())
	}

	requestData := &DoctorTreatmentPlanRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		return 0, nil, errors.New("Unable to parse input parameters: " + err.Error())

	}

	patientVisitId, err := strconv.ParseInt(requestData.PatientVisitId, 10, 64)
	if err != nil {
		return 0, nil, errors.New("Unable to parse patient visit id: " + err.Error())
	}

	return patientVisitId, requestData, nil

}

func (d *DoctorTreatmentPlanHandler) getTreatmentPlanForPatientVisit(w http.ResponseWriter, r *http.Request) {
	patientVisitId, requestData, err := getPatientVisitIdFromRequest(r)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisitReviewData, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	drTreatmentPlan, err := d.DataApi.GetAbbreviatedTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, patientVisitId)
	if err == api.NoRowsError {
		WriteDeveloperError(w, http.StatusNotFound, "No treatment plan exists for patient visit")
		return
	} else if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for patient visit: "+err.Error())
		return
	}

	// only return the small amount of information retreived about the treatment plan
	if requestData.Abbreviated {
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
		return
	}

	drTreatmentPlan.TreatmentList = &common.TreatmentList{}
	drTreatmentPlan.TreatmentList.Treatments, err = d.DataApi.GetTreatmentsBasedOnTreatmentPlanId(patientVisitId, drTreatmentPlan.Id.Int64())
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, http.StatusInternalServerError, "Unable to get treatments for treatment plan: "+err.Error())
		return
	}

	drTreatmentPlan.RegimenPlan, err = d.DataApi.GetRegimenPlanForTreatmentPlan(drTreatmentPlan.Id.Int64())
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, http.StatusInternalServerError, "Unable to get regimen plan for treatment plan: "+err.Error())
		return
	}

	drTreatmentPlan.RegimenPlan.AllRegimenSteps, err = d.DataApi.GetRegimenStepsForDoctor(patientVisitReviewData.DoctorId)
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, http.StatusInternalServerError, "Unable to get all regimen steps for doctor")
		return
	}

	drTreatmentPlan.Advice = &common.Advice{
		TreatmentPlanId: drTreatmentPlan.Id,
		PatientVisitId:  encoding.NewObjectId(patientVisitId),
	}

	drTreatmentPlan.Advice.SelectedAdvicePoints, err = d.DataApi.GetAdvicePointsForTreatmentPlan(drTreatmentPlan.Id.Int64())
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, http.StatusInternalServerError, "Unable to get advice points for treatment plan")
		return
	}

	drTreatmentPlan.Advice.AllAdvicePoints, err = d.DataApi.GetAdvicePointsForDoctor(patientVisitReviewData.DoctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice points for doctor")
		return
	}

	setCommittedStateForEachSection(drTreatmentPlan)

	if err := d.populateFavoriteTreatmentPlanIntoTreatmentPlan(drTreatmentPlan, drTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64()); err == api.NoRowsError {
		WriteDeveloperError(w, http.StatusNotFound, "Favorite treatment plan not found")
		return
	} else if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plan "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
}

func (d *DoctorTreatmentPlanHandler) pickATreatmentPlan(w http.ResponseWriter, r *http.Request) {
	patientVisitId, requestData, err := getPatientVisitIdFromRequest(r)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	var favoriteTreatmentPlanId int64
	if requestData.DoctorFavoriteTreatmentPlanId != "" {
		favoriteTreatmentPlanId, err = strconv.ParseInt(requestData.DoctorFavoriteTreatmentPlanId, 10, 64)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse favorite treatment plan id: "+err.Error())
			return
		}
	}

	patientVisitReviewData, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	patientVisitStatus := patientVisitReviewData.PatientVisit.Status
	if patientVisitStatus != api.CASE_STATUS_REVIEWING && patientVisitStatus != api.CASE_STATUS_SUBMITTED {
		WriteDeveloperError(w, http.StatusForbidden, fmt.Sprintf("Unable to start a new treatment plan for a patient visit that is in the %s state", patientVisitReviewData.PatientVisit.Status))
		return
	}

	// Start new treatment plan for patient visit (indicate favorite treatment plan if indicated)
	// Note that this method deletes any pre-existing treatment plan
	treatmentPlanId, err := d.DataApi.StartNewTreatmentPlanForPatientVisit(patientVisitReviewData.PatientVisit.PatientId.Int64(),
		patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId, favoriteTreatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to start new treatment plan for patient visit: "+err.Error())
		return
	}

	// populate the regimen steps and the advice steps for now until we figure out a cleaner way to handle
	// the client getting the master list of regimen and advice
	allRegimenSteps, err := d.DataApi.GetRegimenStepsForDoctor(patientVisitReviewData.DoctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get regimen steps for doctor: "+err.Error())
		return
	}

	allAdvicePoints, err := d.DataApi.GetAdvicePointsForDoctor(patientVisitReviewData.DoctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice poitns for doctor: "+err.Error())
		return
	}

	drTreatmentPlan := &common.DoctorTreatmentPlan{
		Id:             encoding.NewObjectId(treatmentPlanId),
		PatientVisitId: patientVisitReviewData.PatientVisit.PatientVisitId,
		Advice: &common.Advice{
			AllAdvicePoints: allAdvicePoints,
		},
		RegimenPlan: &common.RegimenPlan{
			AllRegimenSteps: allRegimenSteps,
		},
		TreatmentList: &common.TreatmentList{},
	}

	setCommittedStateForEachSection(drTreatmentPlan)

	if err := d.populateFavoriteTreatmentPlanIntoTreatmentPlan(drTreatmentPlan, favoriteTreatmentPlanId); err == api.NoRowsError {
		WriteDeveloperError(w, http.StatusNotFound, "No favorite treatment plan found")
		return
	} else if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plan: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
}

func setCommittedStateForEachSection(drTreatmentPlan *common.DoctorTreatmentPlan) {
	// depending on which sections have data in them, mark them to be committed or uncommitted
	// note that we intentionally treat a section with no data to be in the UNCOMMITTED state so as
	// to ensure that the doctor actually wanted to leave a particular section blank
	drTreatmentPlan.TreatmentList.Status = api.STATUS_UNCOMMITTED
	drTreatmentPlan.RegimenPlan.Status = api.STATUS_UNCOMMITTED
	drTreatmentPlan.Advice.Status = api.STATUS_UNCOMMITTED
	if len(drTreatmentPlan.TreatmentList.Treatments) > 0 {
		drTreatmentPlan.TreatmentList.Status = api.STATUS_COMMITTED
	}
	if len(drTreatmentPlan.RegimenPlan.RegimenSections) > 0 {
		drTreatmentPlan.RegimenPlan.Status = api.STATUS_COMMITTED
	}
	if len(drTreatmentPlan.Advice.SelectedAdvicePoints) > 0 {
		drTreatmentPlan.Advice.Status = api.STATUS_COMMITTED
	}
}

func (d *DoctorTreatmentPlanHandler) populateFavoriteTreatmentPlanIntoTreatmentPlan(treatmentPlan *common.DoctorTreatmentPlan, favoriteTreatmentPlanId int64) error {
	// Populate treatment plan with favorite treatment plan data if favorite treatment plan specified
	if favoriteTreatmentPlanId == 0 {
		return nil
	}

	favoriteTreatmentPlan, err := d.DataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId)
	if err != nil {
		return err
	}
	treatmentPlan.DoctorFavoriteTreatmentPlanName = favoriteTreatmentPlan.Name
	treatmentPlan.DoctorFavoriteTreatmentPlanId = favoriteTreatmentPlan.Id

	// The assumption here is that all components of a treatment plan that are already populated
	// match the items in the favorite treatment plan, if there exists a mapping to indicate that this
	// treatment plan must be filled in from a favorite treatment plan. The reason that we don't just write over
	// the items that do already belong in the treatment plan is to maintain the ids of the items that have been committed
	// to the database as part of the treatment plan.

	// populate treatments
	if len(treatmentPlan.TreatmentList.Treatments) == 0 {

		treatmentPlan.TreatmentList.Treatments = make([]*common.Treatment, len(favoriteTreatmentPlan.TreatmentList.Treatments))
		for i, treatment := range favoriteTreatmentPlan.TreatmentList.Treatments {
			treatmentPlan.TreatmentList.Treatments[i] = &common.Treatment{
				DrugDBIds:               treatment.DrugDBIds,
				DrugInternalName:        treatment.DrugInternalName,
				DrugName:                treatment.DrugName,
				DrugRoute:               treatment.DrugRoute,
				DosageStrength:          treatment.DosageStrength,
				DispenseValue:           treatment.DispenseValue,
				DispenseUnitId:          treatment.DispenseUnitId,
				DispenseUnitDescription: treatment.DispenseUnitDescription,
				NumberRefills:           treatment.NumberRefills,
				SubstitutionsAllowed:    treatment.SubstitutionsAllowed,
				DaysSupply:              treatment.DaysSupply,
				PharmacyNotes:           treatment.PharmacyNotes,
				PatientInstructions:     treatment.PatientInstructions,
				CreationDate:            treatment.CreationDate,
				OTC:                     treatment.OTC,
				IsControlledSubstance:    treatment.IsControlledSubstance,
				SupplementalInstructions: treatment.SupplementalInstructions,
			}
		}
	}

	// populate regimen plan
	if len(treatmentPlan.RegimenPlan.RegimenSections) == 0 {
		treatmentPlan.RegimenPlan.RegimenSections = make([]*common.RegimenSection, len(favoriteTreatmentPlan.RegimenPlan.RegimenSections))

		for i, regimenSection := range favoriteTreatmentPlan.RegimenPlan.RegimenSections {
			treatmentPlan.RegimenPlan.RegimenSections[i] = &common.RegimenSection{
				RegimenName:  regimenSection.RegimenName,
				RegimenSteps: make([]*common.DoctorInstructionItem, len(regimenSection.RegimenSteps)),
			}

			for j, regimenStep := range regimenSection.RegimenSteps {
				regimenSection.RegimenSteps[j] = &common.DoctorInstructionItem{
					ParentId: regimenStep.ParentId,
					Text:     regimenStep.Text,
				}
			}
		}
	}

	// populate advice
	if len(treatmentPlan.Advice.SelectedAdvicePoints) == 0 {
		treatmentPlan.Advice.SelectedAdvicePoints = make([]*common.DoctorInstructionItem, len(favoriteTreatmentPlan.Advice.SelectedAdvicePoints))
		for i, advicePoint := range favoriteTreatmentPlan.Advice.SelectedAdvicePoints {
			treatmentPlan.Advice.SelectedAdvicePoints[i] = &common.DoctorInstructionItem{
				ParentId: advicePoint.ParentId,
				Text:     advicePoint.Text,
			}
		}
	}

	return nil

}

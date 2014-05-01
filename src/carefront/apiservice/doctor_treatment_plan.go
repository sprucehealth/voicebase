package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type DoctorPickTreatmentPlanHandler struct {
	DataApi api.DataAPI
}

type DoctorPickTreatmentPlanRequestData struct {
	DoctorFavoriteTreatmentPlanId string `schema:"dr_favorite_treatment_plan_id"`
	PatientVisitId                string `schema:"patient_visit_id,required"`
}

type DoctorTreatmentPlan struct {
	Id                              encoding.ObjectId   `json:"id,omitempty"`
	PatientVisitId                  encoding.ObjectId   `json:"patient_visit_id,omitempty"`
	DoctorFavoriteTreatmentPlanId   encoding.ObjectId   `json:"dr_favorite_treatment_plan_id"`
	DoctorFavoriteTreatmentPlanName string              `json:"dr_favorite_treatment_plan_name,omitempty"`
	TreatmentList                   *treatmentList      `json:"treatment_list"`
	RegimenPlan                     *common.RegimenPlan `json:"regimen_plan,omitempty"`
	Advice                          *common.Advice      `json:"advice,omitempty"`
}

type treatmentList struct {
	Treatments []*common.Treatment `json:"treatments,omitempty"`
	Status     string              `json:"status,omitempty"`
}

type DoctorPickTreatmentPlanResponseData struct {
	TreatmentPlan *DoctorTreatmentPlan `json:"treatment_plan"`
}

func (d *DoctorPickTreatmentPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_PUT {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := DoctorPickTreatmentPlanRequestData{}
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientVisitId, err := strconv.ParseInt(requestData.PatientVisitId, 10, 64)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse patient visit id: "+err.Error())
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

	drTreatmentPlan := &DoctorTreatmentPlan{
		Id:             encoding.NewObjectId(treatmentPlanId),
		PatientVisitId: patientVisitReviewData.PatientVisit.PatientVisitId,
	}

	// Populate treatment plan with favorite treatment plan data if favorite treatment plan specified
	if favoriteTreatmentPlanId != 0 {
		favoriteTreatmentPlan, err := d.DataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId)
		if err == api.NoRowsError {
			WriteDeveloperError(w, http.StatusNotFound, "Favorite treatment plan not found")
			return
		} else if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plan "+err.Error())
			return
		}
		populateFavoriteTreatmentPlanIntoTreatmentPlan(drTreatmentPlan, favoriteTreatmentPlan)
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPickTreatmentPlanResponseData{TreatmentPlan: drTreatmentPlan})
}

func populateFavoriteTreatmentPlanIntoTreatmentPlan(treatmentPlan *DoctorTreatmentPlan, favoriteTreatmentPlan *common.FavoriteTreatmentPlan) {

	treatmentPlan.DoctorFavoriteTreatmentPlanName = favoriteTreatmentPlan.Name
	treatmentPlan.DoctorFavoriteTreatmentPlanId = favoriteTreatmentPlan.Id

	// for each of the sections populated from the faovrite treatment plan,
	// indicate to the client that the section has not yet been committed.
	// Doing so enables the client to distinguish between a comitted
	// (where the doctor has explicitly pressed the add button) and non-comitted state

	// The assumption here is that all components of a treatment plan that are already populated
	// match the items in the favorite treatment plan, if there exists a mapping to indicate that this
	// treatment plan must be filled in from a favorite treatment plan. The reason that we don't just write over
	// the items that do already belong in the treatment plan is to maintain the ids of the items that have been committed
	// to the database as part of the treatment plan.

	// populate treatments
	if treatmentPlan.TreatmentList == nil {
		treatmentPlan.TreatmentList = &treatmentList{
			Status: api.STATUS_UNCOMMITTED,
		}

		treatmentPlan.TreatmentList.Treatments = make([]*common.Treatment, len(favoriteTreatmentPlan.Treatments))
		for i, treatment := range favoriteTreatmentPlan.Treatments {
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
	if treatmentPlan.RegimenPlan == nil {
		treatmentPlan.RegimenPlan = &common.RegimenPlan{
			RegimenSections: make([]*common.RegimenSection, len(favoriteTreatmentPlan.RegimenPlan.RegimenSections)),
			Status:          api.STATUS_UNCOMMITTED,
		}
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
	if treatmentPlan.Advice == nil {
		treatmentPlan.Advice = &common.Advice{
			SelectedAdvicePoints: make([]*common.DoctorInstructionItem, len(favoriteTreatmentPlan.Advice.SelectedAdvicePoints)),
			Status:               api.STATUS_UNCOMMITTED,
		}
		for i, advicePoint := range favoriteTreatmentPlan.Advice.SelectedAdvicePoints {
			treatmentPlan.Advice.SelectedAdvicePoints[i] = &common.DoctorInstructionItem{
				ParentId: advicePoint.ParentId,
				Text:     advicePoint.Text,
			}
		}
	}

}

package doctor_treatment_plan

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/dispatch"
	"fmt"
	"net/http"
)

type doctorTreatmentPlanHandler struct {
	dataApi api.DataAPI
}

func NewDoctorTreatmentPlanHandler(dataApi api.DataAPI) *doctorTreatmentPlanHandler {
	return &doctorTreatmentPlanHandler{
		dataApi: dataApi,
	}
}

type DoctorTreatmentPlanRequestData struct {
	DoctorFavoriteTreatmentPlanId int64 `schema:"dr_favorite_treatment_plan_id"`
	TreatmentPlanId               int64 `schema:"treatment_plan_id"`
	PatientVisitId                int64 `schema:"patient_visit_id"`
	Abbreviated                   bool  `schema:"abbreviated"`
}

type DoctorTreatmentPlanResponse struct {
	TreatmentPlan *common.DoctorTreatmentPlan `json:"treatment_plan"`
}

func (d *doctorTreatmentPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		d.getTreatmentPlan(w, r)
	case apiservice.HTTP_PUT:
		d.pickATreatmentPlan(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *doctorTreatmentPlanHandler) getTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	requestData := &DoctorTreatmentPlanRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	if requestData.TreatmentPlanId == 0 {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "treatment_plan_id not specified")
		return
	}

	doctorId, err := d.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	drTreatmentPlan, err := d.dataApi.GetAbridgedTreatmentPlan(requestData.TreatmentPlanId, doctorId)
	if err == api.NoRowsError {
		http.NotFound(w, r)
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan for patient visit: "+err.Error())
		return
	}

	// only return the small amount of information retreived about the treatment plan
	if requestData.Abbreviated {
		apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
		return
	}

	if err := fillInTreatmentPlan(drTreatmentPlan, d.dataApi); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	drTreatmentPlan.RegimenPlan.AllRegimenSteps, err = d.dataApi.GetRegimenStepsForDoctor(doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get all regimen steps for doctor")
		return
	}

	drTreatmentPlan.Advice.AllAdvicePoints, err = d.dataApi.GetAdvicePointsForDoctor(doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice points for doctor")
		return
	}

	setCommittedStateForEachSection(drTreatmentPlan)

	if err := d.populateFavoriteTreatmentPlanIntoTreatmentPlan(drTreatmentPlan, drTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64()); err == api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusNotFound, "Favorite treatment plan not found")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plan "+err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
}

func (d *doctorTreatmentPlanHandler) pickATreatmentPlan(w http.ResponseWriter, r *http.Request) {
	requestData := &DoctorTreatmentPlanRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	} else if requestData.PatientVisitId == 0 {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "PatientVisitId not specified")
		return
	}

	patientVisitReviewData, statusCode, err := apiservice.ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, apiservice.GetContext(r).AccountId, d.dataApi)
	if err != nil {
		apiservice.WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	patientVisitStatus := patientVisitReviewData.PatientVisit.Status
	if patientVisitStatus != api.CASE_STATUS_REVIEWING && patientVisitStatus != api.CASE_STATUS_SUBMITTED {
		apiservice.WriteDeveloperError(w, http.StatusForbidden, fmt.Sprintf("Unable to start a new treatment plan for a patient visit that is in the %s state", patientVisitReviewData.PatientVisit.Status))
		return
	}

	// Start new treatment plan for patient visit (indicate favorite treatment plan if indicated)
	// Note that this method deletes any pre-existing treatment plan
	treatmentPlanId, err := d.dataApi.StartNewTreatmentPlanForPatientVisit(patientVisitReviewData.PatientVisit.PatientId.Int64(),
		patientVisitReviewData.PatientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId, requestData.DoctorFavoriteTreatmentPlanId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to start new treatment plan for patient visit: "+err.Error())
		return
	}

	// populate the regimen steps and the advice steps for now until we figure out a cleaner way to handle
	// the client getting the master list of regimen and advice
	allRegimenSteps, err := d.dataApi.GetRegimenStepsForDoctor(patientVisitReviewData.DoctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get regimen steps for doctor: "+err.Error())
		return
	}

	allAdvicePoints, err := d.dataApi.GetAdvicePointsForDoctor(patientVisitReviewData.DoctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice poitns for doctor: "+err.Error())
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
		Status:        api.STATUS_DRAFT,
	}

	setCommittedStateForEachSection(drTreatmentPlan)

	if err := d.populateFavoriteTreatmentPlanIntoTreatmentPlan(drTreatmentPlan, requestData.DoctorFavoriteTreatmentPlanId); err == api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusNotFound, "No favorite treatment plan found")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plan: "+err.Error())
		return
	}

	dispatch.Default.Publish(&NewTreatmentPlanStartedEvent{
		DoctorId:        patientVisitReviewData.DoctorId,
		PatientVisitId:  requestData.PatientVisitId,
		TreatmentPlanId: treatmentPlanId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
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

func fillInTreatmentPlan(drTreatmentPlan *common.DoctorTreatmentPlan, dataApi api.DataAPI) error {
	var err error
	drTreatmentPlan.TreatmentList = &common.TreatmentList{}
	drTreatmentPlan.TreatmentList.Treatments, err = dataApi.GetTreatmentsBasedOnTreatmentPlanId(drTreatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get treatments for treatment plan: %s", err)
	}

	drTreatmentPlan.RegimenPlan, err = dataApi.GetRegimenPlanForTreatmentPlan(drTreatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get regimen plan for treatment plan: %s", err)
	}

	drTreatmentPlan.Advice = &common.Advice{
		TreatmentPlanId: drTreatmentPlan.Id,
		PatientVisitId:  drTreatmentPlan.PatientVisitId,
	}

	drTreatmentPlan.Advice.SelectedAdvicePoints, err = dataApi.GetAdvicePointsForTreatmentPlan(drTreatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get advice points for treatment plan")
	}
	return err
}

func (d *doctorTreatmentPlanHandler) populateFavoriteTreatmentPlanIntoTreatmentPlan(treatmentPlan *common.DoctorTreatmentPlan, favoriteTreatmentPlanId int64) error {
	// Populate treatment plan with favorite treatment plan data if favorite treatment plan specified
	if favoriteTreatmentPlanId == 0 {
		return nil
	}

	favoriteTreatmentPlan, err := d.dataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId)
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
				treatmentPlan.RegimenPlan.RegimenSections[i].RegimenSteps[j] = &common.DoctorInstructionItem{
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

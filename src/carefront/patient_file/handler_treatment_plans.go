package patient_file

// import (
// 	"carefront/api"
// 	"carefront/apiservice"
// 	"carefront/common"
// 	"fmt"
// 	"net/http"

// 	"github.com/gorilla/schema"
// )

// type treatmentPlansHandler struct {
// 	dataApi api.DataAPI
// }

// type treatmentPlansRequestData struct {
// 	PatientId       int64 `schema:"patient_id"`
// 	Preview         bool  `schema:"preview"`
// 	TreatmentPlanId int64 `schema:"treatment_plan_id"`
// }

// type treatmentPlansResponseData struct {
// 	ActiveTreatmentPlans  []*common.DoctorTreatmentPlan `json:"active_treatment_plans,omitempty"`
// 	InActiveTreatmentPlan []*common.DoctorTreatmentPlan `json:"inactive_treatment_plans,omitempty"`
// }

// func NewTreatmentPlansHandler(dataApi api.DataAPI) {
// 	return *treatmentPlansHandler{
// 		dataApi: dataApi,
// 	}
// }

// func (t *treatmentPlansHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	if err := r.ParseForm(); err != nil {
// 		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
// 		return
// 	}

// 	requestData := treatmentPlansRequestData{}
// 	if err := DecodeRequestData(requestData interface{}, r *http.Request); err != nil {
// 		apiservice.Wri
// 	}
// 	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
// 		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
// 		return
// 	}

// }

// func (t *treatmentPlansHandler) getTreatmentPlans(requestData *treatmentPlansRequestData, w http.ResponseWriter, r *http.Request) {

// 	// send a list of treatment plans for the patient
// 	if requestData.PatientId != 0 {
// 		treatmentPlans, err := t.dataApi.GetAbridgedTreatmentPlanListForPatient(requestData.PatientId)
// 		if err != nil {
// 			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plans for patient: "+err.Error())
// 			return
// 		}

// 		apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &treatmentPlansRequestData{
// 			ActiveTreatmentPlans: treatmentPlans,
// 		})
// 		return
// 	}

// 	// ensure that treatment plan id is set
// 	if requestData.TreatmentPlanId == 0 {
// 		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Treatment plan id or patient id expected to be specified")
// 		return
// 	}

// 	drTreatmentPlan, err := t.dataApi.GetAbridgedTreatmentPlanForDoctor(requestData.TreatmentPlanId)
// 	if err == api.NoRowsError {
// 		http.NotFound(w, r)
// 		return
// 	} else if err != nil {
// 		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "unable to treatment plan by id: "+err.Error())
// 		return
// 	}

// 	if err := fillInTreatmentPlan(drTreatmentPlan, t.dataApi); err != nil {
// 		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to fill in treatment plan: "+err.Error())
// 		return
// 	}

// 	if err != nil {
// 		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get all regimen steps for doctor")
// 		return
// 	}

// 	drTreatmentPlan.Advice.AllAdvicePoints, err = d.dataApi.GetAdvicePointsForDoctor(patientVisitReviewData.DoctorId)
// 	if err != nil {
// 		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice points for doctor")
// 		return
// 	}

// 	setCommittedStateForEachSection(drTreatmentPlan)

// 	if err := d.populateFavoriteTreatmentPlanIntoTreatmentPlan(drTreatmentPlan, drTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64()); err == api.NoRowsError {
// 		apiservice.WriteDeveloperError(w, http.StatusNotFound, "Favorite treatment plan not found")
// 		return
// 	} else if err != nil {
// 		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatment plan "+err.Error())
// 		return
// 	}

// 	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorTreatmentPlanResponse{TreatmentPlan: drTreatmentPlan})
// }

// func fillInTreatmentPlan(drTreatmentPlan *common.DoctorTreatmentPlan, dataApi api.DataAPI) error {
// 	var err error
// 	drTreatmentPlan.TreatmentList = &common.TreatmentList{}
// 	drTreatmentPlan.TreatmentList.Treatments, err = dataApi.GetTreatmentsBasedOnTreatmentPlanId(drTreatmentPlan.Id.Int64())
// 	if err != nil {
// 		return fmt.Errorf("Unable to get treatments for treatment plan: %s", err)
// 	}

// 	drTreatmentPlan.RegimenPlan, err = dataApi.GetRegimenPlanForTreatmentPlan(drTreatmentPlan.Id.Int64())
// 	if err != nil {
// 		return fmt.Errorf("Unable to get regimen plan for treatment plan: %s", err)
// 	}

// 	drTreatmentPlan.Advice = &common.Advice{
// 		TreatmentPlanId: drTreatmentPlan.Id,
// 		PatientVisitId:  drTreatmentPlan.PatientVisitId,
// 	}

// 	drTreatmentPlan.Advice.SelectedAdvicePoints, err = dataApi.GetAdvicePointsForTreatmentPlan(drTreatmentPlan.Id.Int64())
// 	if err != nil {
// 		return fmt.Errorf("Unable to get advice points for treatment plan")
// 	}
// 	return err
// }

// func (d *doctorTreatmentPlanHandler) populateFavoriteTreatmentPlanIntoTreatmentPlan(treatmentPlan *common.DoctorTreatmentPlan, favoriteTreatmentPlanId int64) error {
// 	// Populate treatment plan with favorite treatment plan data if favorite treatment plan specified
// 	if favoriteTreatmentPlanId == 0 {
// 		return nil
// 	}

// 	favoriteTreatmentPlan, err := d.dataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId)
// 	if err != nil {
// 		return err
// 	}
// 	treatmentPlan.DoctorFavoriteTreatmentPlanName = favoriteTreatmentPlan.Name
// 	treatmentPlan.DoctorFavoriteTreatmentPlanId = favoriteTreatmentPlan.Id

// 	// The assumption here is that all components of a treatment plan that are already populated
// 	// match the items in the favorite treatment plan, if there exists a mapping to indicate that this
// 	// treatment plan must be filled in from a favorite treatment plan. The reason that we don't just write over
// 	// the items that do already belong in the treatment plan is to maintain the ids of the items that have been committed
// 	// to the database as part of the treatment plan.

// 	// populate treatments
// 	if len(treatmentPlan.TreatmentList.Treatments) == 0 {

// 		treatmentPlan.TreatmentList.Treatments = make([]*common.Treatment, len(favoriteTreatmentPlan.TreatmentList.Treatments))
// 		for i, treatment := range favoriteTreatmentPlan.TreatmentList.Treatments {
// 			treatmentPlan.TreatmentList.Treatments[i] = &common.Treatment{
// 				DrugDBIds:               treatment.DrugDBIds,
// 				DrugInternalName:        treatment.DrugInternalName,
// 				DrugName:                treatment.DrugName,
// 				DrugRoute:               treatment.DrugRoute,
// 				DosageStrength:          treatment.DosageStrength,
// 				DispenseValue:           treatment.DispenseValue,
// 				DispenseUnitId:          treatment.DispenseUnitId,
// 				DispenseUnitDescription: treatment.DispenseUnitDescription,
// 				NumberRefills:           treatment.NumberRefills,
// 				SubstitutionsAllowed:    treatment.SubstitutionsAllowed,
// 				DaysSupply:              treatment.DaysSupply,
// 				PharmacyNotes:           treatment.PharmacyNotes,
// 				PatientInstructions:     treatment.PatientInstructions,
// 				CreationDate:            treatment.CreationDate,
// 				OTC:                     treatment.OTC,
// 				IsControlledSubstance:    treatment.IsControlledSubstance,
// 				SupplementalInstructions: treatment.SupplementalInstructions,
// 			}
// 		}
// 	}

// 	// populate regimen plan
// 	if len(treatmentPlan.RegimenPlan.RegimenSections) == 0 {
// 		treatmentPlan.RegimenPlan.RegimenSections = make([]*common.RegimenSection, len(favoriteTreatmentPlan.RegimenPlan.RegimenSections))

// 		for i, regimenSection := range favoriteTreatmentPlan.RegimenPlan.RegimenSections {
// 			treatmentPlan.RegimenPlan.RegimenSections[i] = &common.RegimenSection{
// 				RegimenName:  regimenSection.RegimenName,
// 				RegimenSteps: make([]*common.DoctorInstructionItem, len(regimenSection.RegimenSteps)),
// 			}

// 			for j, regimenStep := range regimenSection.RegimenSteps {
// 				treatmentPlan.RegimenPlan.RegimenSections[i].RegimenSteps[j] = &common.DoctorInstructionItem{
// 					ParentId: regimenStep.ParentId,
// 					Text:     regimenStep.Text,
// 				}
// 			}
// 		}
// 	}

// 	// populate advice
// 	if len(treatmentPlan.Advice.SelectedAdvicePoints) == 0 {
// 		treatmentPlan.Advice.SelectedAdvicePoints = make([]*common.DoctorInstructionItem, len(favoriteTreatmentPlan.Advice.SelectedAdvicePoints))
// 		for i, advicePoint := range favoriteTreatmentPlan.Advice.SelectedAdvicePoints {
// 			treatmentPlan.Advice.SelectedAdvicePoints[i] = &common.DoctorInstructionItem{
// 				ParentId: advicePoint.ParentId,
// 				Text:     advicePoint.Text,
// 			}
// 		}
// 	}

// 	return nil

// }

// func setCommittedStateForEachSection(drTreatmentPlan *common.DoctorTreatmentPlan) {
// 	// depending on which sections have data in them, mark them to be committed or uncommitted
// 	// note that we intentionally treat a section with no data to be in the UNCOMMITTED state so as
// 	// to ensure that the doctor actually wanted to leave a particular section blank
// 	drTreatmentPlan.TreatmentList.Status = api.STATUS_UNCOMMITTED
// 	drTreatmentPlan.RegimenPlan.Status = api.STATUS_UNCOMMITTED
// 	drTreatmentPlan.Advice.Status = api.STATUS_UNCOMMITTED
// 	if len(drTreatmentPlan.TreatmentList.Treatments) > 0 {
// 		drTreatmentPlan.TreatmentList.Status = api.STATUS_COMMITTED
// 	}
// 	if len(drTreatmentPlan.RegimenPlan.RegimenSections) > 0 {
// 		drTreatmentPlan.RegimenPlan.Status = api.STATUS_COMMITTED
// 	}
// 	if len(drTreatmentPlan.Advice.SelectedAdvicePoints) > 0 {
// 		drTreatmentPlan.Advice.Status = api.STATUS_COMMITTED
// 	}
// }

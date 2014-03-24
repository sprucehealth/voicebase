package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/schema"
)

type TreatmentsHandler struct {
	DataApi api.DataAPI
	ErxApi  erx.ERxAPI
}

type GetTreatmentsResponse struct {
	Treatments []*common.Treatment `json:"treatments"`
}

type AddTreatmentsResponse struct {
	TreatmentIds []string `json:"treatment_ids"`
}

type AddTreatmentsRequestBody struct {
	Treatments     []*common.Treatment `json:"treatments"`
	PatientVisitId *common.ObjectId    `json:"patient_visit_id"`
}

type GetTreatmentsRequestBody struct {
	PatientVisitId  int64 `schema:"patient_visit_id"`
	TreatmentPlanId int64 `schema:"treatment_plan_id"`
}

func (t *TreatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_GET:
		t.getTreatments(w, r)
	case HTTP_POST:
		t.addTreatment(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (t *TreatmentsHandler) getTreatments(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData GetTreatmentsRequestBody
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	if err := ensureTreatmentPlanOrPatientVisitIdPresent(t.DataApi, treatmentPlanId, &patientVisitId); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisitReviewData, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, GetContext(r).AccountId, t.DataApi)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, "Doctor not authorized to get treatments for patient visit: "+err.Error())
		return
	}

	if treatmentPlanId == 0 {
		treatmentPlanId, err = t.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, patientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get active treatment plan id based on patient visit id : "+err.Error())
			return
		}
	}

	treatments, err := t.DataApi.GetTreatmentsBasedOnTreatmentPlanId(patientVisitId, treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "unable to get treatments for patient visit : "+err.Error())
		return
	}

	// for each of the drugs, go ahead and fill in the drug name, route and form
	for _, treatment := range treatments {
		if treatment.DrugForm == "" {
			drugName, drugForm, drugRoute := breakDrugInternalNameIntoComponents(treatment.DrugInternalName)
			treatment.DrugName = drugName
			// only break down name into route and form if the route and form are non-empty strings
			if drugForm != "" && drugRoute != "" {
				treatment.DrugForm = drugForm
				treatment.DrugRoute = drugRoute
			}
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{Treatments: treatments})
}

func (t *TreatmentsHandler) addTreatment(w http.ResponseWriter, r *http.Request) {
	treatmentsRequestBody := &AddTreatmentsRequestBody{}

	if err := json.NewDecoder(r.Body).Decode(treatmentsRequestBody); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatment body: "+err.Error())
		return
	}

	if treatmentsRequestBody.Treatments == nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Nothing to do becuase no treatments were passed to add ")
		return
	}

	if treatmentsRequestBody.PatientVisitId.Int64() == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "Patient visit id must be specified")
		return
	}

	patientVisitReviewData, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(treatmentsRequestBody.PatientVisitId.Int64(), GetContext(r).AccountId, t.DataApi)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, "Unable to validate doctor to add treatment to patient visit: "+err.Error())
		return
	}

	// intentionally not requiring the treatment plan id from the client when adding treatments because it should only be possible to
	// add treatments to an active treatment plan
	treatmentPlanId, err := t.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, treatmentsRequestBody.PatientVisitId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the treatment plan from the patient visit: "+err.Error())
		return
	}

	if err := EnsurePatientVisitInExpectedStatus(t.DataApi, treatmentsRequestBody.PatientVisitId.Int64(), api.CASE_STATUS_REVIEWING); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	doctor, err := t.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	//  validate all treatments
	for _, treatment := range treatmentsRequestBody.Treatments {
		err = validateTreatment(treatment)
		if err != nil {
			WriteUserError(w, http.StatusBadRequest, err.Error())
			return
		}

		// break up the name into its components so that it can be saved into the database as its components
		drugName, drugForm, drugRoute := breakDrugInternalNameIntoComponents(treatment.DrugInternalName)
		treatment.DrugName = drugName
		// only break down name into route and form if the route and form are non-empty strings
		if drugForm != "" && drugRoute != "" {
			treatment.DrugForm = drugForm
			treatment.DrugRoute = drugRoute
		}

		if treatment.DoctorTreatmentTemplateId.Int64() != 0 {
			// check to ensure that the drug is still in market; we do so by ensuring that we are still able
			// to get back the drug db ids to identify this drug
			medicationToCheck, err := t.ErxApi.SelectMedication(doctor.DoseSpotClinicianId, treatment.DrugInternalName, treatment.DosageStrength)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to select medication to identify whether or not it is still available in the market: "+err.Error())
				return
			}

			// if not, we cannot allow the doctor to prescribe this drug given that its no longer in market (a surescripts requirement)
			if medicationToCheck == nil {
				WriteUserError(w, http.StatusBadRequest, fmt.Sprintf("%s %s is no longer available and cannot be prescribed to the patient. We suggest that you remove this saved template from your list.", treatment.DrugInternalName, treatment.DosageStrength))
				return
			}
		}
	}

	// Add treatments to patient
	if err := t.DataApi.AddTreatmentsForPatientVisit(treatmentsRequestBody.Treatments, patientVisitReviewData.DoctorId, treatmentPlanId); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment to patient visit: "+err.Error())
		return
	}

	treatments, err := t.DataApi.GetTreatmentsBasedOnTreatmentPlanId(treatmentsRequestBody.PatientVisitId.Int64(), treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "unable to get treatments for patient visit after adding treatments : "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{Treatments: treatments})
}

func validateTreatment(treatment *common.Treatment) error {
	if treatment.DrugInternalName == "" {
		return errors.New("Drug Internal name for treatment cannot be empty")
	}

	if treatment.DosageStrength == "" {
		return errors.New("Dosage Strength for treatment cannot be empty")
	}

	if treatment.DispenseValue == 0 {
		return errors.New("DispenseValue for treatment cannot be 0")
	}

	if treatment.DispenseUnitId.Int64() == 0 {
		return errors.New("DispenseUnit	 Id for treatment cannot be 0")
	}

	if treatment.NumberRefills == 0 {
		return errors.New("Number of refills for treatment cannot be 0")
	}

	if treatment.DaysSupply == 0 {
		return errors.New("Days of Supply for treatment cannot be 0")
	}

	if treatment.PatientInstructions == "" {
		return errors.New("Patient Instructions for treatment cannot be empty")
	}

	if treatment.DrugDBIds == nil || len(treatment.DrugDBIds) == 0 {
		return errors.New("Drug DB Ids for treatment cannot be empty")
	}
	return nil
}

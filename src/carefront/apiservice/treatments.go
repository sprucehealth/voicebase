package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"errors"
	"github.com/gorilla/schema"
	"net/http"
)

type TreatmentsHandler struct {
	DataApi api.DataAPI
}

type GetTreatmentsResponse struct {
	Treatments []*common.Treatment `json:"treatments"`
}

type AddTreatmentsResponse struct {
	TreatmentIds []string `json:"treatment_ids"`
}

type AddTreatmentsRequestBody struct {
	Treatments     []*common.Treatment `json:"treatments"`
	PatientVisitId int64               `json:"patient_visit_id,string"`
}

type GetTreatmentsRequestBody struct {
	PatientVisitId  int64 `schema:"patient_visit_id"`
	TreatmentPlanId int64 `schema:"treatment_plan_id"`
}

func NewTreatmentsHandler(dataApi api.DataAPI) *TreatmentsHandler {
	return &TreatmentsHandler{DataApi: dataApi}
}

func (t *TreatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		t.getTreatments(w, r)
	case "POST":
		t.addTreatment(w, r)
	}
}

func (t *TreatmentsHandler) getTreatments(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(GetTreatmentsRequestBody)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	err = ensureTreatmentPlanOrPatientVisitIdPresent(t.DataApi, &treatmentPlanId, &patientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	doctorId, _, _, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, GetContext(r).AccountId, t.DataApi)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, "Doctor not authorized to get treatments for patient visit: "+err.Error())
		return
	}

	if treatmentPlanId == 0 {
		treatmentPlanId, err = t.DataApi.GetActiveTreatmentPlanForPatientVisit(doctorId, patientVisitId)
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
	jsonDecoder := json.NewDecoder(r.Body)
	treatmentsRequestBody := &AddTreatmentsRequestBody{}

	err := jsonDecoder.Decode(treatmentsRequestBody)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatment body: "+err.Error())
		return
	}

	if treatmentsRequestBody.Treatments == nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Nothing to do becuase no treatments were passed to add ")
		return
	}

	if treatmentsRequestBody.PatientVisitId == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "Patient visit id must be specified: "+err.Error())
		return
	}

	doctorId, _, _, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(treatmentsRequestBody.PatientVisitId, GetContext(r).AccountId, t.DataApi)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, "Unable to validate doctor to add treatment to patient visit: "+err.Error())
		return
	}

	// intentionally not requiring the treatment plan id from the client when adding treatments because it should only be possible to
	// add treatments to an active treatment plan
	treatmentPlanId, err := t.DataApi.GetActiveTreatmentPlanForPatientVisit(doctorId, treatmentsRequestBody.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the treatment plan from the patient visit: "+err.Error())
		return
	}

	err = EnsurePatientVisitInExpectedStatus(t.DataApi, treatmentsRequestBody.PatientVisitId, api.CASE_STATUS_REVIEWING)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	//  validate all treatments
	for _, treatment := range treatmentsRequestBody.Treatments {
		err = validateTreatment(treatment)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, err.Error())
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
	}

	// Add treatments to patient
	err = t.DataApi.AddTreatmentsForPatientVisit(treatmentsRequestBody.Treatments, doctorId, treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment to patient visit: "+err.Error())
		return
	}

	treatments, err := t.DataApi.GetTreatmentsBasedOnTreatmentPlanId(treatmentsRequestBody.PatientVisitId, treatmentPlanId)
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

	if treatment.DispenseUnitId == 0 {
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

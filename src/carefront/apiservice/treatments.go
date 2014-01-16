package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"github.com/gorilla/schema"
	"net/http"
	"strconv"
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
	PatientVisitId int64 `schema:"patient_visit_id"`
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

	_, _, _, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, GetContext(r).AccountId, t.DataApi)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, "Doctor not authorized to get treatments for patient visit: "+err.Error())
		return
	}

	treatmentPlan, err := t.DataApi.GetTreatmentPlanForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "unable to get treatments for patient visit : "+err.Error())
		return
	}

	if treatmentPlan == nil {
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{Treatments: nil})
		return
	}

	// for each of the drugs, go ahead and fill in the drug name, route and form
	for _, treatment := range treatmentPlan.Treatments {
		drugName, drugForm, drugRoute := breakDrugInternalNameIntoComponents(treatment.DrugInternalName)
		treatment.DrugName = drugName
		// only break down name into route and form if the route and form are non-empty strings
		if drugForm != "" && drugRoute != "" {
			treatment.DrugForm = drugForm
			treatment.DrugRoute = drugRoute
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{Treatments: treatmentPlan.Treatments})
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

	_, _, _, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(treatmentsRequestBody.PatientVisitId, GetContext(r).AccountId, t.DataApi)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, "Unable to validate doctor to add treatment to patient visit: "+err.Error())
		return
	}

	err = EnsurePatientVisitInExpectedStatus(t.DataApi, treatmentsRequestBody.PatientVisitId, api.CASE_STATUS_REVIEWING)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	//  validate all treatments
	for _, treatment := range treatmentsRequestBody.Treatments {
		if treatment.DrugInternalName == "" {
			WriteDeveloperError(w, http.StatusBadRequest, "Drug Internal name for treatment cannot be empty")
			return
		}

		if treatment.DosageStrength == "" {
			WriteDeveloperError(w, http.StatusBadRequest, "Dosage Strength for treatment cannot be empty")
			return
		}

		if treatment.DispenseValue == 0 {
			WriteDeveloperError(w, http.StatusBadRequest, "DispenseValue for treatment cannot be 0")
			return
		}

		if treatment.DispenseUnitId == 0 {
			WriteDeveloperError(w, http.StatusBadRequest, "DispenseUnitId for treatment cannot be 0")
			return
		}

		if treatment.NumberRefills == 0 {
			WriteDeveloperError(w, http.StatusBadRequest, "Number of refills for treatment cannot be 0")
			return
		}

		if treatment.DaysSupply == 0 {
			WriteDeveloperError(w, http.StatusBadRequest, "Days of Supply for treatment cannot be 0")
			return
		}

		if treatment.PatientInstructions == "" {
			WriteUserError(w, http.StatusBadRequest, "Patient Instructions for treatment cannot be empty")
			return
		}

		if treatment.DrugDBIds == nil || len(treatment.DrugDBIds) == 0 {
			WriteDeveloperError(w, http.StatusBadRequest, "Drug DB Ids for treatment cannot be empty")
			return
		}
	}

	// Add treatments to patient
	err = t.DataApi.AddTreatmentsForPatientVisit(treatmentsRequestBody.Treatments, treatmentsRequestBody.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment to patient visit: "+err.Error())
		return
	}

	treatmentIds := make([]string, 0)
	for _, treatment := range treatmentsRequestBody.Treatments {
		treatmentIds = append(treatmentIds, strconv.FormatInt(treatment.Id, 10))
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &AddTreatmentsResponse{TreatmentIds: treatmentIds})
}

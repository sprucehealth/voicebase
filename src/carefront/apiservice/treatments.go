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
	DataApi   api.DataAPI
	accountId int64
}

type GetTreatmentsResponse struct {
	Treatments []*common.Treatment `json:"treatments"`
}

type AddTreatmentsResponse struct {
	TreatmentIds []string `json:"treatment_ids"`
}

type TreatmentsRequestBody struct {
	Treatments     []*common.Treatment `json:"treatments"`
	PatientVisitId int64               `schema:"patient_visit_id"`
}

func NewTreatmentsHandler(dataApi api.DataAPI) *TreatmentsHandler {
	return &TreatmentsHandler{dataApi, 0}
}

func (t *TreatmentsHandler) AccountIdFromAuthToken(accountId int64) {
	t.accountId = accountId
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
	requestData := new(TreatmentsRequestBody)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	_, _, _, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, t.accountId, t.DataApi)
	if err != nil {
		WriteDeveloperError(w, httpStatusCode, "Doctor not authorized to get treatments for patient visit: "+err.Error())
		return
	}

	treatmentPlan, err := t.DataApi.GetTreatmentPlanForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "unable to get treatments for patient visit : "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{Treatments: treatmentPlan.Treatments})

}

func (t *TreatmentsHandler) addTreatment(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	treatmentsRequestBody := &TreatmentsRequestBody{}

	err := jsonDecoder.Decode(treatmentsRequestBody)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatment body: "+err.Error())
		return
	}

	if len(treatmentsRequestBody.Treatments) == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "Nothing to do becuase no treatments were passed to add: "+err.Error())
		return
	}

	// just to be on the safe side, verify each of the treatments that the doctor is trying to add
	for _, treatment := range treatmentsRequestBody.Treatments {
		_, _, _, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(treatment.PatientVisitId, t.accountId, t.DataApi)
		if err != nil {
			WriteDeveloperError(w, httpStatusCode, "Unable to validate doctor to add treatment to patient visit: "+err.Error())
			return
		}
	}

	// TODO  validate all treatments
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
			WriteDeveloperError(w, http.StatusBadRequest, "Patient Instructions for treatment cannot be empty")
			return
		}
	}

	// Add treatments to patient
	err = t.DataApi.AddTreatmentsForPatientVisit(treatmentsRequestBody.Treatments)
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

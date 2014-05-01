package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/dispatch"
	"carefront/libs/erx"
	"encoding/json"
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
	PatientVisitId encoding.ObjectId   `json:"patient_visit_id"`
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
		treatment.DrugName, treatment.DrugForm, treatment.DrugRoute = breakDrugInternalNameIntoComponents(treatment.DrugInternalName)
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
		if err := validateTreatment(treatment); err != nil {
			WriteUserError(w, http.StatusBadRequest, err.Error())
			return
		}

		// break up the name into its components so that it can be saved into the database as its components
		treatment.DrugName, treatment.DrugForm, treatment.DrugRoute = breakDrugInternalNameIntoComponents(treatment.DrugInternalName)

		httpStatusCode, errorResponse := checkIfDrugInTreatmentFromTemplateIsOutOfMarket(treatment, doctor, t.ErxApi)
		if errorResponse != nil {
			WriteError(w, httpStatusCode, *errorResponse)
			return
		}

	}

	// Add treatments to patient
	if err := t.DataApi.AddTreatmentsForPatientVisit(treatmentsRequestBody.Treatments, patientVisitReviewData.DoctorId, treatmentPlanId, patientVisitReviewData.PatientVisit.PatientId.Int64()); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment to patient visit: "+err.Error())
		return
	}

	treatments, err := t.DataApi.GetTreatmentsBasedOnTreatmentPlanId(treatmentsRequestBody.PatientVisitId.Int64(), treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "unable to get treatments for patient visit after adding treatments : "+err.Error())
		return
	}

	dispatch.Default.Publish(&TreatmentsAddedEvent{
		PatientVisitId: patientVisitReviewData.PatientVisit.PatientVisitId.Int64(),
		DoctorId:       doctor.DoctorId.Int64(),
		Treatments:     treatments,
	})

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{Treatments: treatments})
}

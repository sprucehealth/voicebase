package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/dispatch"
	"carefront/libs/erx"
	"encoding/json"
	"net/http"
)

type TreatmentsHandler struct {
	DataApi api.DataAPI
	ErxApi  erx.ERxAPI
}

type GetTreatmentsResponse struct {
	TreatmentList *common.TreatmentList `json:"treatment_list"`
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
	case HTTP_POST:
		t.addTreatment(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
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
		treatment.DrugName, treatment.DrugForm, treatment.DrugRoute = BreakDrugInternalNameIntoComponents(treatment.DrugInternalName)

		httpStatusCode, errorResponse := checkIfDrugInTreatmentFromTemplateIsOutOfMarket(treatment, doctor, t.ErxApi)
		if errorResponse != nil {
			WriteErrorResponse(w, httpStatusCode, *errorResponse)
			return
		}

	}

	// Add treatments to patient
	if err := t.DataApi.AddTreatmentsForPatientVisit(treatmentsRequestBody.Treatments, patientVisitReviewData.DoctorId, treatmentPlanId, patientVisitReviewData.PatientVisit.PatientId.Int64()); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment to patient visit: "+err.Error())
		return
	}

	treatments, err := t.DataApi.GetTreatmentsBasedOnTreatmentPlanId(treatmentPlanId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "unable to get treatments for patient visit after adding treatments : "+err.Error())
		return
	}

	dispatch.Default.PublishAsync(&TreatmentsAddedEvent{
		TreatmentPlanId: treatmentPlanId,
		DoctorId:        doctor.DoctorId.Int64(),
		Treatments:      treatments,
	})

	treatmentList := &common.TreatmentList{
		Treatments: treatments,
		Status:     api.STATUS_COMMITTED,
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{TreatmentList: treatmentList})
}

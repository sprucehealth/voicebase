package doctor_treatment_plan

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/dispatch"
	"carefront/libs/erx"
	"net/http"
)

type treatmentsHandler struct {
	dataAPI api.DataAPI
	erxAPI  erx.ERxAPI
}

func NewTreatmentsHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI) *treatmentsHandler {
	return &treatmentsHandler{
		dataAPI: dataAPI,
		erxAPI:  erxAPI,
	}
}

type GetTreatmentsResponse struct {
	TreatmentList *common.TreatmentList `json:"treatment_list"`
}

type AddTreatmentsResponse struct {
	TreatmentIds []string `json:"treatment_ids"`
}

type AddTreatmentsRequestBody struct {
	Treatments      []*common.Treatment `json:"treatments"`
	TreatmentPlanId encoding.ObjectId   `json:"treatment_plan_id"`
}

func (t *treatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_POST:
		t.addTreatment(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (t *treatmentsHandler) addTreatment(w http.ResponseWriter, r *http.Request) {
	treatmentsRequestBody := &AddTreatmentsRequestBody{}
	if err := apiservice.DecodeRequestData(treatmentsRequestBody, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if treatmentsRequestBody.TreatmentPlanId.Int64() == 0 {
		apiservice.WriteValidationError("treatment_plan_id must be specified", w, r)
		return
	}

	if treatmentsRequestBody.Treatments == nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Nothing to do becuase no treatments were passed to add ")
		return
	}

	patientVisitId, err := t.dataAPI.GetPatientVisitIdFromTreatmentPlanId(treatmentsRequestBody.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientVisitReviewData, httpStatusCode, err := apiservice.ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisitId, apiservice.GetContext(r).AccountId, t.dataAPI)
	if err != nil {
		apiservice.WriteDeveloperError(w, httpStatusCode, "Unable to validate doctor to add treatment to patient visit: "+err.Error())
		return
	}

	if err := apiservice.EnsurePatientVisitInExpectedStatus(t.dataAPI, patientVisitId, api.CASE_STATUS_REVIEWING); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	doctor, err := t.dataAPI.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	//  validate all treatments
	for _, treatment := range treatmentsRequestBody.Treatments {
		if err := apiservice.ValidateTreatment(treatment); err != nil {
			apiservice.WriteUserError(w, http.StatusBadRequest, err.Error())
			return
		}

		// break up the name into its components so that it can be saved into the database as its components
		treatment.DrugName, treatment.DrugForm, treatment.DrugRoute = apiservice.BreakDrugInternalNameIntoComponents(treatment.DrugInternalName)

		httpStatusCode, errorResponse := apiservice.CheckIfDrugInTreatmentFromTemplateIsOutOfMarket(treatment, doctor, t.erxAPI)
		if errorResponse != nil {
			apiservice.WriteErrorResponse(w, httpStatusCode, *errorResponse)
			return
		}

	}

	// Add treatments to patient
	if err := t.dataAPI.AddTreatmentsForPatientVisit(treatmentsRequestBody.Treatments, patientVisitReviewData.DoctorId, treatmentsRequestBody.TreatmentPlanId.Int64(), patientVisitReviewData.PatientVisit.PatientId.Int64()); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment to patient visit: "+err.Error())
		return
	}

	treatments, err := t.dataAPI.GetTreatmentsBasedOnTreatmentPlanId(treatmentsRequestBody.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "unable to get treatments for patient visit after adding treatments : "+err.Error())
		return
	}

	dispatch.Default.PublishAsync(&TreatmentsAddedEvent{
		TreatmentPlanId: treatmentsRequestBody.TreatmentPlanId.Int64(),
		DoctorId:        doctor.DoctorId.Int64(),
		Treatments:      treatments,
	})

	treatmentList := &common.TreatmentList{
		Treatments: treatments,
		Status:     api.STATUS_COMMITTED,
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{TreatmentList: treatmentList})
}

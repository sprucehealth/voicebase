package doctor_treatment_plan

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
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

	patientId, err := t.dataAPI.GetPatientIdFromTreatmentPlanId(treatmentsRequestBody.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctor, err := t.dataAPI.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	treatmentPlan, err := t.dataAPI.GetAbridgedTreatmentPlan(treatmentsRequestBody.TreatmentPlanId.Int64(), doctor.DoctorId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if treatmentPlan.Status != api.STATUS_DRAFT {
		apiservice.WriteValidationError("treatment plan must be in draft mode", w, r)
		return
	}

	if err := apiservice.ValidateWriteAccessToPatientCase(doctor.DoctorId.Int64(), patientId, treatmentPlan.PatientCaseId.Int64(), t.dataAPI); err != nil {
		apiservice.WriteError(err, w, r)
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
	if err := t.dataAPI.AddTreatmentsForTreatmentPlan(treatmentsRequestBody.Treatments, doctor.DoctorId.Int64(), treatmentsRequestBody.TreatmentPlanId.Int64(), patientId); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment to patient visit: "+err.Error())
		return
	}

	treatments, err := t.dataAPI.GetTreatmentsBasedOnTreatmentPlanId(treatmentsRequestBody.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "unable to get treatments for patient visit after adding treatments : "+err.Error())
		return
	}

	dispatch.Default.Publish(&TreatmentsAddedEvent{
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

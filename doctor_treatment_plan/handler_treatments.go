package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
)

type treatmentsHandler struct {
	dataAPI    api.DataAPI
	erxAPI     erx.ERxAPI
	dispatcher *dispatch.Dispatcher
}

func NewTreatmentsHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return &treatmentsHandler{
		dataAPI:    dataAPI,
		erxAPI:     erxAPI,
		dispatcher: dispatcher,
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
	TreatmentPlanID encoding.ObjectId   `json:"treatment_plan_id"`
}

func (t *treatmentsHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	requestData := &AddTreatmentsRequestBody{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	} else if requestData.TreatmentPlanID.Int64() == 0 {
		return false, apiservice.NewValidationError("treatment_plan_id must be specified", r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctor, err := t.dataAPI.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	treatmentPlan, err := t.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID.Int64(), doctor.DoctorId.Int64())
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctor.DoctorId.Int64(), treatmentPlan.PatientId, treatmentPlan.PatientCaseId.Int64(), t.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (t *treatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*AddTreatmentsRequestBody)
	doctor := ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.DoctorTreatmentPlan)

	if requestData.Treatments == nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Nothing to do becuase no treatments were passed to add ")
		return
	}

	if !treatmentPlan.InDraftMode() {
		apiservice.WriteValidationError("treatment plan must be in draft mode", w, r)
		return
	}

	//  validate all treatments
	for _, treatment := range requestData.Treatments {
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
	if err := t.dataAPI.AddTreatmentsForTreatmentPlan(requestData.Treatments, doctor.DoctorId.Int64(), requestData.TreatmentPlanID.Int64(), treatmentPlan.PatientId); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment to patient visit: "+err.Error())
		return
	}

	treatments, err := t.dataAPI.GetTreatmentsBasedOnTreatmentPlanId(requestData.TreatmentPlanID.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "unable to get treatments for patient visit after adding treatments : "+err.Error())
		return
	}

	t.dispatcher.Publish(&TreatmentsAddedEvent{
		TreatmentPlanID: requestData.TreatmentPlanID.Int64(),
		DoctorId:        doctor.DoctorId.Int64(),
		Treatments:      treatments,
	})

	treatmentList := &common.TreatmentList{
		Treatments: treatments,
		Status:     api.STATUS_COMMITTED,
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{TreatmentList: treatmentList})
}

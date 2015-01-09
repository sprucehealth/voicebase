package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type treatmentsHandler struct {
	dataAPI    api.DataAPI
	erxAPI     erx.ERxAPI
	dispatcher *dispatch.Dispatcher
}

func NewTreatmentsHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&treatmentsHandler{
			dataAPI:    dataAPI,
			erxAPI:     erxAPI,
			dispatcher: dispatcher,
		}), []string{"POST"})
}

type GetTreatmentsResponse struct {
	TreatmentList *common.TreatmentList `json:"treatment_list"`
}

type AddTreatmentsResponse struct {
	TreatmentIDs []string `json:"treatment_ids"`
}

type AddTreatmentsRequestBody struct {
	Treatments      []*common.Treatment `json:"treatments"`
	TreatmentPlanID encoding.ObjectID   `json:"treatment_plan_id"`
}

func (t *treatmentsHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	requestData := &AddTreatmentsRequestBody{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if requestData.TreatmentPlanID.Int64() == 0 {
		return false, apiservice.NewValidationError("treatment_plan_id must be specified")
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctor, err := t.dataAPI.GetDoctorFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	treatmentPlan, err := t.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID.Int64(), doctor.DoctorID.Int64())
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctor.DoctorID.Int64(), treatmentPlan.PatientID, treatmentPlan.PatientCaseID.Int64(), t.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (t *treatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*AddTreatmentsRequestBody)
	doctor := ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

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
		var form, route string
		treatment.DrugName, form, route = apiservice.BreakDrugInternalNameIntoComponents(treatment.DrugInternalName)
		if form != "" {
			treatment.DrugForm = form
		}
		if route != "" {
			treatment.DrugRoute = route
		}

		httpStatusCode, errorResponse := apiservice.CheckIfDrugInTreatmentFromTemplateIsOutOfMarket(treatment, doctor, t.erxAPI)
		if errorResponse != nil {
			apiservice.WriteErrorResponse(w, httpStatusCode, *errorResponse)
			return
		}

	}

	// Add treatments to patient
	if err := t.dataAPI.AddTreatmentsForTreatmentPlan(requestData.Treatments, doctor.DoctorID.Int64(), requestData.TreatmentPlanID.Int64(), treatmentPlan.PatientID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatments, err := t.dataAPI.GetTreatmentsBasedOnTreatmentPlanID(requestData.TreatmentPlanID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	t.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		SectionUpdated:  TreatmentsSection,
		DoctorID:        doctor.DoctorID.Int64(),
		TreatmentPlanID: treatmentPlan.ID.Int64(),
	})

	treatmentList := &common.TreatmentList{
		Treatments: treatments,
		Status:     api.STATUS_COMMITTED,
	}

	if err := indicateExistenceOfRXGuidesForTreatments(t.dataAPI, treatmentList); err != nil {
		golog.Errorf(err.Error())
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetTreatmentsResponse{TreatmentList: treatmentList})
}

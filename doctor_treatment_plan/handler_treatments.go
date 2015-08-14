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
	"golang.org/x/net/context"
)

type treatmentsHandler struct {
	dataAPI    api.DataAPI
	erxAPI     erx.ERxAPI
	dispatcher *dispatch.Dispatcher
}

func NewTreatmentsHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI, dispatcher *dispatch.Dispatcher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&treatmentsHandler{
				dataAPI:    dataAPI,
				erxAPI:     erxAPI,
				dispatcher: dispatcher,
			})),
		httputil.Post)
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

func (t *treatmentsHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(ctx)
	requestCache := apiservice.MustCtxCache(ctx)
	if account.Role != api.RoleDoctor {
		return false, apiservice.NewAccessForbiddenError()
	}

	requestData := &AddTreatmentsRequestBody{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if requestData.TreatmentPlanID.Int64() == 0 {
		return false, apiservice.NewValidationError("treatment_plan_id must be specified")
	}
	requestCache[apiservice.CKRequestData] = requestData

	doctor, err := t.dataAPI.GetDoctorFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctor] = doctor

	treatmentPlan, err := t.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID.Int64(), doctor.ID.Int64())
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKTreatmentPlan] = treatmentPlan

	if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctor.ID.Int64(), treatmentPlan.PatientID, treatmentPlan.PatientCaseID.Int64(), t.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (t *treatmentsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	requestData := requestCache[apiservice.CKRequestData].(*AddTreatmentsRequestBody)
	doctor := requestCache[apiservice.CKDoctor].(*common.Doctor)
	treatmentPlan := requestCache[apiservice.CKTreatmentPlan].(*common.TreatmentPlan)

	if !treatmentPlan.InDraftMode() {
		apiservice.WriteValidationError(ctx, "treatment plan must be in draft mode", w, r)
		return
	}

	if err := validateTreatments(
		requestData.Treatments,
		t.dataAPI,
		t.erxAPI,
		doctor.DoseSpotClinicianID); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// FIXME: Remove the validation to ensure drugs are in market until we have come up with a long
	// term solution of how to deal with saved treatments and treatments in FTPs. The problem right now
	// is that the drug search via drug name and dosage strength is brittle in that the name/dosage strength
	// can change.
	// If the drug is not available then it will be rejected at the time of attempting to send the prescriptions
	// which appens after the doctor has submitted the treatmetn plan which makes it an operational problem
	// and not a doctor/patient facing problem so we should be in the clear.

	// Add treatments to patient
	if err := t.dataAPI.AddTreatmentsForTreatmentPlan(requestData.Treatments,
		doctor.ID.Int64(),
		requestData.TreatmentPlanID.Int64(),
		treatmentPlan.PatientID); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	treatments, err := t.dataAPI.GetTreatmentsBasedOnTreatmentPlanID(requestData.TreatmentPlanID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	t.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		SectionUpdated:  TreatmentsSection,
		DoctorID:        doctor.ID.Int64(),
		TreatmentPlanID: treatmentPlan.ID.Int64(),
	})

	treatmentList := &common.TreatmentList{
		Treatments: treatments,
		Status:     api.StatusCommitted,
	}

	if err := indicateExistenceOfRXGuidesForTreatments(t.dataAPI, treatmentList); err != nil {
		golog.Errorf(err.Error())
	}

	httputil.JSONResponse(w, http.StatusOK, &GetTreatmentsResponse{TreatmentList: treatmentList})
}

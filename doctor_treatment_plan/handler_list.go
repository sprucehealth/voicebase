package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type listHandler struct {
	dataAPI api.DataAPI
}

type listHandlerRequestData struct {
	PatientID common.PatientID `schema:"patient_id"`
}

type TreatmentPlansResponse struct {
	DraftTreatmentPlans    []*common.TreatmentPlan `json:"draft_treatment_plans,omitempty"`
	ActiveTreatmentPlans   []*common.TreatmentPlan `json:"active_treatment_plans,omitempty"`
	InactiveTreatmentPlans []*common.TreatmentPlan `json:"inactive_treatment_plans,omitempty"`
}

func NewDeprecatedListHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&listHandler{
				dataAPI: dataAPI,
			})),
		httputil.Get)
}

func (l *listHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	requestData := &listHandlerRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = requestData

	if !requestData.PatientID.IsValid {
		return false, apiservice.NewValidationError("patient_id required")
	}

	doctorID, err := l.dataAPI.GetDoctorIDFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctorID] = doctorID

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, account.Role, doctorID, requestData.PatientID, l.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (l *listHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	requestData := requestCache[apiservice.CKRequestData].(*listHandlerRequestData)

	// NOTE this API is deprecated and only used on the doctor app pre-BL in production.
	// We assume here that the patient has just a single case
	cases, err := l.dataAPI.GetCasesForPatient(requestData.PatientID, []string{common.PCStatusInactive.String(), common.PCStatusActive.String()})
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	} else if len(cases) == 0 {
		apiservice.WriteResourceNotFoundError(ctx, "no cases exist for patient", w, r)
		return
	}

	activeTreatmentPlans, err := l.dataAPI.GetAbridgedTreatmentPlanList(doctorID, cases[0].ID.Int64(), common.ActiveTreatmentPlanStates())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	inactiveTreatmentPlans, err := l.dataAPI.GetAbridgedTreatmentPlanList(doctorID, cases[0].ID.Int64(), []common.TreatmentPlanStatus{common.TPStatusInactive})
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	draftTreatmentPlans, err := l.dataAPI.GetAbridgedTreatmentPlanListInDraftForDoctor(doctorID, cases[0].ID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &TreatmentPlansResponse{
		DraftTreatmentPlans:    draftTreatmentPlans,
		ActiveTreatmentPlans:   activeTreatmentPlans,
		InactiveTreatmentPlans: inactiveTreatmentPlans,
	})
}

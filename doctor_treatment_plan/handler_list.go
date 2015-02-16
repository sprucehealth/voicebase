package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type listHandler struct {
	dataAPI api.DataAPI
}

type listHandlerRequestData struct {
	PatientID int64 `schema:"patient_id"`
}

type TreatmentPlansResponse struct {
	DraftTreatmentPlans    []*common.TreatmentPlan `json:"draft_treatment_plans,omitempty"`
	ActiveTreatmentPlans   []*common.TreatmentPlan `json:"active_treatment_plans,omitempty"`
	InactiveTreatmentPlans []*common.TreatmentPlan `json:"inactive_treatment_plans,omitempty"`
}

func NewDeprecatedListHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&listHandler{
			dataAPI: dataAPI,
		}), []string{"GET"})
}

func (l *listHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &listHandlerRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	if requestData.PatientID == 0 {
		return false, apiservice.NewValidationError("PatientId required")
	}

	doctorID, err := l.dataAPI.GetDoctorIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctorID, requestData.PatientID, l.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (l *listHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*listHandlerRequestData)

	// NOTE this API is deprecated and only used on the doctor app pre-BL in production.
	// We assume here that the patient has just a single case
	cases, err := l.dataAPI.GetCasesForPatient(requestData.PatientID, []string{common.PCStatusInactive.String(), common.PCStatusActive.String()})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if len(cases) == 0 {
		apiservice.WriteResourceNotFoundError("no cases exist for patient", w, r)
		return
	}

	activeTreatmentPlans, err := l.dataAPI.GetAbridgedTreatmentPlanList(doctorID, cases[0].ID.Int64(), common.ActiveTreatmentPlanStates())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	inactiveTreatmentPlans, err := l.dataAPI.GetAbridgedTreatmentPlanList(doctorID, cases[0].ID.Int64(), []common.TreatmentPlanStatus{common.TPStatusInactive})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	draftTreatmentPlans, err := l.dataAPI.GetAbridgedTreatmentPlanListInDraftForDoctor(doctorID, cases[0].ID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &TreatmentPlansResponse{
		DraftTreatmentPlans:    draftTreatmentPlans,
		ActiveTreatmentPlans:   activeTreatmentPlans,
		InactiveTreatmentPlans: inactiveTreatmentPlans,
	})
}

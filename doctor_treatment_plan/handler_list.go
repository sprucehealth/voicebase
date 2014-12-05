package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type listHandler struct {
	dataApi api.DataAPI
}

type listHandlerRequestData struct {
	PatientId int64 `schema:"patient_id"`
}

type TreatmentPlansResponse struct {
	DraftTreatmentPlans    []*common.TreatmentPlan `json:"draft_treatment_plans,omitempty"`
	ActiveTreatmentPlans   []*common.TreatmentPlan `json:"active_treatment_plans,omitempty"`
	InactiveTreatmentPlans []*common.TreatmentPlan `json:"inactive_treatment_plans,omitempty"`
}

func NewListHandler(dataApi api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&listHandler{
			dataApi: dataApi,
		}), []string{"GET"})
}

func (l *listHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &listHandlerRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	if requestData.PatientId == 0 {
		return false, apiservice.NewValidationError("PatientId required", r)
	}

	doctorId, err := l.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorId

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctorId, requestData.PatientId, l.dataApi); err != nil {
		return false, err
	}

	return true, nil
}

func (l *listHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	doctorId := ctxt.RequestCache[apiservice.DoctorID].(int64)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*listHandlerRequestData)

	activeTreatmentPlans, err := l.dataApi.GetAbridgedTreatmentPlanList(doctorId, requestData.PatientId, common.ActiveTreatmentPlanStates())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	inactiveTreatmentPlans, err := l.dataApi.GetAbridgedTreatmentPlanList(doctorId, requestData.PatientId, []common.TreatmentPlanStatus{common.TPStatusInactive})
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	draftTreatmentPlans, err := l.dataApi.GetAbridgedTreatmentPlanListInDraftForDoctor(doctorId, requestData.PatientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &TreatmentPlansResponse{
		DraftTreatmentPlans:    draftTreatmentPlans,
		ActiveTreatmentPlans:   activeTreatmentPlans,
		InactiveTreatmentPlans: inactiveTreatmentPlans,
	})
}

package treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type treatmentsHandler struct {
	dataAPI api.DataAPI
}

type treatmentsViewsResponse struct {
	TreatmentViews []tpView `json:"treatment_views"`
}

func NewTreatmentsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&treatmentsHandler{
				dataAPI: dataAPI,
			}), []string{"GET"})
}

func (t *treatmentsHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (t *treatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientId, err := t.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	tps, err := t.dataAPI.GetActiveTreatmentPlansForPatient(patientId)
	if err == api.NoRowsError || (err == nil && len(tps) == 0) {
		apiservice.WriteResourceNotFoundError("No treatment plan found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// TODO: For now just use the first since that's all there should be. When multiple
	// conditions are supported this should merge all treatments in some way.
	treatmentPlan := tps[0]

	treatmentPlan.TreatmentList = &common.TreatmentList{}
	treatmentPlan.TreatmentList.Treatments, err = t.dataAPI.GetTreatmentsBasedOnTreatmentPlanId(treatmentPlan.Id.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctor, err := t.dataAPI.GetDoctorFromId(treatmentPlan.DoctorId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	views := generateViewsForTreatments(treatmentPlan, doctor, t.dataAPI, true)
	for _, v := range views {
		if err := v.Validate(); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSON(w, &treatmentsViewsResponse{
		TreatmentViews: views,
	})
}

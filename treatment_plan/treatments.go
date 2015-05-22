package treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/views"
)

type treatmentsHandler struct {
	dataAPI api.DataAPI
}

type treatmentsViewsResponse struct {
	TreatmentViews []views.View `json:"treatment_views"`
}

func NewTreatmentsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&treatmentsHandler{
				dataAPI: dataAPI,
			}), httputil.Get)
}

func (t *treatmentsHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.RolePatient {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (t *treatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientID, err := t.dataAPI.GetPatientIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	tps, err := t.dataAPI.GetActiveTreatmentPlansForPatient(patientID)
	if api.IsErrNotFound(err) || (err == nil && len(tps) == 0) {
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
	treatmentPlan.TreatmentList.Treatments, err = t.dataAPI.GetTreatmentsBasedOnTreatmentPlanID(treatmentPlan.ID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	tViews := GenerateViewsForTreatments(treatmentPlan.TreatmentList, treatmentPlan.ID.Int64(), t.dataAPI, true)
	if err := views.Validate(tViews, treatmentViewNamespace); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &treatmentsViewsResponse{
		TreatmentViews: tViews,
	})
}

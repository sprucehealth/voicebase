package treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
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

func NewTreatmentsHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&treatmentsHandler{
				dataAPI: dataAPI,
			}),
			api.RoleDoctor),
		httputil.Get)

}

func (t *treatmentsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)
	patientID, err := t.dataAPI.GetPatientIDFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	tps, err := t.dataAPI.GetActiveTreatmentPlansForPatient(patientID)
	if api.IsErrNotFound(err) || (err == nil && len(tps) == 0) {
		apiservice.WriteResourceNotFoundError(ctx, "No treatment plan found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// TODO: For now just use the first since that's all there should be. When multiple
	// conditions are supported this should merge all treatments in some way.
	treatmentPlan := tps[0]

	treatmentPlan.TreatmentList = &common.TreatmentList{}
	treatmentPlan.TreatmentList.Treatments, err = t.dataAPI.GetTreatmentsBasedOnTreatmentPlanID(treatmentPlan.ID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	tViews := GenerateViewsForTreatments(treatmentPlan.TreatmentList, treatmentPlan.ID.Int64(), t.dataAPI, true)
	if err := views.Validate(tViews, treatmentViewNamespace); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &treatmentsViewsResponse{
		TreatmentViews: tViews,
	})
}

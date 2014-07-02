package treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type treatmentsHandler struct {
	dataAPI api.DataAPI
}

type treatmentsViewsResponse struct {
	TreatmentViews []tpView `json:"treatment_views"`
}

func NewTreatmentsHandler(dataAPI api.DataAPI) *treatmentsHandler {
	return &treatmentsHandler{
		dataAPI: dataAPI,
	}
}

func (t *treatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	patientId, err := t.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatmentPlan, err := t.dataAPI.GetActiveTreatmentPlanForPatient(patientId)
	if err == api.NoRowsError {
		apiservice.WriteResourceNotFoundError("No treatment plan found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

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

	views := generateViewsForTreatments(treatmentPlan.TreatmentList, doctor, t.dataAPI, true)
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

package patient_file

import (
	"carefront/api"
	"carefront/common"
	"net/http"
)

type listHandler struct {
	dataApi api.DataAPI
}

type listHandlerRequestData struct {
	PatientId int64 `schema:"patient_id"`
}

type treatmentPlansResponseData struct {
	DraftTreatmentPlans   []*common.DoctorTreatmentPlan `json:"draft_treatment_plans,omitempty"`
	ActiveTreatmentPlans  []*common.DoctorTreatmentPlan `json:"active_treatment_plans,omitempty"`
	InActiveTreatmentPlan []*common.DoctorTreatmentPlan `json:"inactive_treatment_plans,omitempty"`
}

func NewListTreatmentPlansHandler(dataApi api.DataAPI) *listHandler {
	return *listHandler{
		dataApi: dataApi,
	}
}

func (l *listHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestData := &listHandlerRequestData{}
	if err := DecodeRequestData(requestData, r); err != nil {

	}

}

package demo

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

type demoVisitHandler struct {
	dataAPI api.DataAPI
}

func NewTrainingCasesHandler(dataAPI api.DataAPI) http.Handler {
	return &demoVisitHandler{
		dataAPI: dataAPI,
	}
}

func (d *demoVisitHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	switch ctxt.Role {
	case api.DOCTOR_ROLE:
	default:
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (d *demoVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := d.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := d.dataAPI.ClaimTrainingSet(doctorID, apiservice.HEALTH_CONDITION_ACNE_ID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

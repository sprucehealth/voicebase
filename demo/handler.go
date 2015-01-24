package demo

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type demoVisitHandler struct {
	dataAPI api.DataAPI
}

func NewTrainingCasesHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&demoVisitHandler{
					dataAPI: dataAPI,
				}), []string{api.DOCTOR_ROLE}),
		[]string{"POST"})
}

func (d *demoVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := d.dataAPI.GetDoctorIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// TODO: don't assume acne
	if err := d.dataAPI.ClaimTrainingSet(doctorID, api.AcnePathwayTag); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

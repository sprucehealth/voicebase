package demo

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
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
				}), api.RoleDoctor),
		httputil.Post)
}

func (d *demoVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(r.Context())
	doctorID, err := d.dataAPI.GetDoctorIDFromAccountID(account.ID)
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

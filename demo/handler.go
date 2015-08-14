package demo

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type demoVisitHandler struct {
	dataAPI api.DataAPI
}

func NewTrainingCasesHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&demoVisitHandler{
					dataAPI: dataAPI,
				}), api.RoleDoctor),
		httputil.Post)
}

func (d *demoVisitHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)
	doctorID, err := d.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// TODO: don't assume acne
	if err := d.dataAPI.ClaimTrainingSet(doctorID, api.AcnePathwayTag); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

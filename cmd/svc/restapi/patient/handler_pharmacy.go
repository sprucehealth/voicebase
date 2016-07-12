package patient

import (
	"net/http"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/pharmacy"
	"github.com/sprucehealth/backend/libs/httputil"
)

type pharmacyHandler struct {
	dataAPI api.DataAPI
}

func NewPharmacyHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&pharmacyHandler{
				dataAPI: dataAPI,
			}), api.RolePatient), httputil.Post)
}

func (u *pharmacyHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var pharmacy pharmacy.PharmacyData
	if err := apiservice.DecodeRequestData(&pharmacy, r); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	patient, err := u.dataAPI.GetPatientFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if err := u.dataAPI.UpdatePatientPharmacy(patient.ID, &pharmacy); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

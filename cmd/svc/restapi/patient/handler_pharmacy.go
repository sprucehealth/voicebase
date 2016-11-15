package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/pharmacy"
)

type pharmacyHandler struct {
	dataAPI api.DataAPI
}

func NewPharmacyHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&pharmacyHandler{
				dataAPI: dataAPI,
			}), api.RolePatient), httputil.Post)
}

func (u *pharmacyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var pharmacy pharmacy.PharmacyData
	if err := apiservice.DecodeRequestData(&pharmacy, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patient, err := u.dataAPI.GetPatientFromAccountID(apiservice.MustCtxAccount(r.Context()).ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := u.dataAPI.UpdatePatientPharmacy(patient.ID, &pharmacy); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/pharmacy"
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

	patient, err := u.dataAPI.GetPatientFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := u.dataAPI.UpdatePatientPharmacy(patient.ID.Int64(), &pharmacy); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

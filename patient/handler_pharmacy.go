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
		apiservice.AuthorizationRequired(&pharmacyHandler{
			dataAPI: dataAPI,
		}), []string{"POST"})
}

func (u *pharmacyHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}
func (u *pharmacyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var pharmacy pharmacy.PharmacyData
	if err := apiservice.DecodeRequestData(&pharmacy, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patient, err := u.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := u.dataAPI.UpdatePatientPharmacy(patient.PatientId.Int64(), &pharmacy); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

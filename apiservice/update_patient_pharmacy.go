package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/pharmacy"
)

type updatePatientPharmacyHandler struct {
	dataAPI api.DataAPI
}

func NewUpdatePatientPharmacyHandler(dataAPI api.DataAPI) http.Handler {
	return &updatePatientPharmacyHandler{
		dataAPI: dataAPI,
	}
}

func (u *updatePatientPharmacyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_POST:
		u.updatePatientPharmacy(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (u *updatePatientPharmacyHandler) updatePatientPharmacy(w http.ResponseWriter, r *http.Request) {
	var pharmacy pharmacy.PharmacyData
	if err := DecodeRequestData(&pharmacy, r); err != nil {
		WriteError(err, w, r)
		return
	}

	patient, err := u.dataAPI.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	if err := u.dataAPI.UpdatePatientPharmacy(patient.PatientId.Int64(), &pharmacy); err != nil {
		WriteError(err, w, r)
		return
	}

	WriteJSONSuccess(w)
}

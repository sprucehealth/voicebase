package apiservice

import (
	"carefront/api"
	"carefront/libs/pharmacy"
	"encoding/json"
	"net/http"
)

type UpdatePatientPharmacyHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

func (u *UpdatePatientPharmacyHandler) AccountIdFromAuthToken(accountId int64) {
	u.accountId = accountId
}

func (u *UpdatePatientPharmacyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		u.updatePatientPharmacy(w, r)
	default:
		WriteJSONToHTTPResponseWriter(w, http.StatusNotFound, nil)
	}
}

func (u *UpdatePatientPharmacyHandler) updatePatientPharmacy(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	pharmacy := &pharmacy.PharmacyData{}

	err := jsonDecoder.Decode(pharmacy)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patient, err := u.DataApi.GetPatientFromAccountId(u.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from account id: "+err.Error())
		return
	}

	err = u.DataApi.UpdatePatientPharmacy(patient.PatientId, pharmacy.Id, pharmacy.Source)
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, http.StatusInternalServerError, "Unable to set the patient pharmacy: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

package apiservice

import (
	"carefront/api"
	"github.com/gorilla/schema"
	"net/http"
)

const (
	oddity_pharmacy_type = "oddity"
)

type UpdatePatientPharmacyHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type UpdatePatientPharmacyRequestData struct {
	PharmacyId int64 `schema:"pharmacy_id,required"`
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
	r.ParseForm()
	requestData := new(UpdatePatientPharmacyRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patient, err := u.DataApi.GetPatientFromAccountId(u.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from account id: "+err.Error())
		return
	}

	err = u.DataApi.UpdatePatientPharmacy(patient.PatientId, requestData.PharmacyId, oddity_pharmacy_type)
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, http.StatusInternalServerError, "Unable to set the patient pharmacy: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

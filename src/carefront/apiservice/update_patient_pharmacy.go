package apiservice

import (
	"carefront/api"
	"carefront/libs/golog"
	"carefront/libs/pharmacy"
	"encoding/json"
	"net/http"
)

type UpdatePatientPharmacyHandler struct {
	DataApi               api.DataAPI
	PharmacySearchService pharmacy.PharmacySearchAPI
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
	var pharmacy pharmacy.PharmacyData
	if err := json.NewDecoder(r.Body).Decode(&pharmacy); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patient, err := u.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from account id: "+err.Error())
		return
	}

	pharmacyDetails, err := u.PharmacySearchService.GetPharmacyBasedOnId(pharmacy.Id)
	pharmacyDetails.Source = pharmacy.Source
	if err != nil {
		golog.Warningf("Unable to get the pharmacy details when it would've been nice to be able to do so: " + err.Error())
	}

	err = u.DataApi.UpdatePatientPharmacy(patient.PatientId, pharmacyDetails)
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, http.StatusInternalServerError, "Unable to set the patient pharmacy: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

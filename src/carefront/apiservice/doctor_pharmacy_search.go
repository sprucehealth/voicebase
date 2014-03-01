package apiservice

import (
	"carefront/api"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"encoding/json"
	"net/http"
)

type DoctorPharmacySearchHandler struct {
	DataApi api.DataAPI
	ErxApi  erx.ERxAPI
}

type DoctorPharmacySearchRequestData struct {
	SearchString  string   `json:"search_string"`
	PharmacyTypes []string `json:"pharmacy_types"`
}

type DoctorPharmacySearchResponse struct {
	PharmacyResults []*pharmacy.PharmacyData `json:"pharmacy_results"`
}

func (d *DoctorPharmacySearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	requestData := &DoctorPharmacySearchRequestData{}
	if err := json.NewDecoder(r.Body).Decode(requestData); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from id: "+err.Error())
		return
	}

	pharmacyResults, err := d.ErxApi.SearchForPharmacies(doctor.DoseSpotClinicianId, "", "", requestData.SearchString, "", requestData.PharmacyTypes)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to search for pharmacies: "+err.Error())
		return
	}
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPharmacySearchResponse{PharmacyResults: pharmacyResults})
}

package apiservice

import (
	"carefront/api"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"net/http"

	"github.com/gorilla/schema"
)

type DoctorPharmacySearchHandler struct {
	DataApi api.DataAPI
	ErxApi  erx.ERxAPI
}

type DoctorPharmacySearchRequestData struct {
	SearchString  string   `schema:"search_string"`
	PharmacyTypes []string `schema:"pharmacy_types"`
}

type DoctorPharmacySearchResponse struct {
	PharmacyResults []*pharmacy.PharmacyData `json:"pharmacy_results"`
}

func (d *DoctorPharmacySearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &DoctorPharmacySearchRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
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

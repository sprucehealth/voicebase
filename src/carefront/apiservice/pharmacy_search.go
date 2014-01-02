package apiservice

import (
	"carefront/api"
	"carefront/libs/maps"
	"github.com/gorilla/schema"
	"net/http"
)

const (
	defaultNumResults          = 10
	defaultSearchRadiusInMiles = 10
)

type PharmacySearchHandler struct {
	PharmacySearchService api.PharmacySearchAPI
	MapsService           maps.MapsService
}

type PharmacySearchRequestData struct {
	NumResults          int64  `schema:"num_results"`
	SearchRadiusInMiles int64  `schema:"search_radius"`
	SearchLocation      string `schema:"search_location,required"`
}

type PharmacySearchResponse struct {
	Pharmacies []*api.PharmacyData `json:"pharmacy_results"`
}

func (p *PharmacySearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(PharmacySearchRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	if requestData.NumResults == 0 {
		requestData.NumResults = defaultNumResults
	}

	if requestData.SearchRadiusInMiles == 0 {
		requestData.SearchRadiusInMiles = defaultSearchRadiusInMiles
	}

	locationInfo, err := p.MapsService.GetLatLongFromSearchLocation(requestData.SearchLocation)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to convert search location to lat,long: "+err.Error())
		return
	}

	pharmacies, err := p.PharmacySearchService.GetPharmaciesAroundSearchLocation(locationInfo.Latitude, locationInfo.Longitude, float64(requestData.SearchRadiusInMiles), requestData.NumResults)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacies based on location: "+err.Error())
		return
	}

	pharmacyResult := &PharmacySearchResponse{}
	pharmacyResult.Pharmacies = pharmacies

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, pharmacyResult)
}

package apiservice

import (
	"net/http"

	"carefront/libs/maps"
	"carefront/libs/pharmacy"
	"github.com/gorilla/schema"
	"github.com/samuel/go-cache/cache"
)

const (
	defaultNumResults          = 10
	defaultSearchRadiusInMiles = 10
)

var locationCache cache.Cache = cache.NewLFUCache(2048)

type PharmacySearchHandler struct {
	PharmacySearchService pharmacy.PharmacySearchAPI
	MapsService           maps.MapsService
}

type PharmacySearchRequestData struct {
	NumResults          int64  `schema:"num_results"`
	SearchRadiusInMiles int64  `schema:"search_radius"`
	SearchLocation      string `schema:"search_location,required"`
}

type PharmacySearchResponse struct {
	Pharmacies []*pharmacy.PharmacyData `json:"pharmacy_results"`
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

	var locationInfo *maps.LocationInfo
	if li, err := locationCache.Get(requestData.SearchLocation); err == nil && li != nil {
		locationInfo = li.(*maps.LocationInfo)
	} else {
		locationInfo, err = p.MapsService.GetLatLongFromSearchLocation(requestData.SearchLocation)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to convert search location to lat,long: "+err.Error())
			return
		}
		locationCache.Set(requestData.SearchLocation, locationInfo)
	}

	var pharmacies []*pharmacy.PharmacyData

	if locationInfo != nil {
		pharmacies, err = p.PharmacySearchService.GetPharmaciesAroundSearchLocation(locationInfo.Latitude, locationInfo.Longitude, float64(requestData.SearchRadiusInMiles), requestData.NumResults)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacies based on location: "+err.Error())
			return
		}
	} else {
		pharmacies = make([]*pharmacy.PharmacyData, 0)
	}

	pharmacyResult := &PharmacySearchResponse{
		Pharmacies: pharmacies,
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, pharmacyResult)
}

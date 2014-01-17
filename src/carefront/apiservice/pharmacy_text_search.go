package apiservice

import (
	"carefront/api"
	"carefront/libs/golog"
	"carefront/libs/maps"
	"carefront/libs/pharmacy"
	"github.com/gorilla/schema"
	"github.com/samuel/go-cache/cache"
	"net/http"
	"strconv"
)

var locationCache cache.Cache = cache.NewLFUCache(2048)

const (
	defaultSearchRadiusInMeters = "1000"
)

type PharmacyTextSearchHandler struct {
	PharmacySearchService pharmacy.PharmacySearchAPI
	MapsService           maps.MapsService
	DataApi               api.DataAPI
}

type PharmacyTextSearchRequestData struct {
	SearchRadiusInMiles string `schema:"search_radius"`
	Latitude            string `schema:"latitude"`
	Longitude           string `schema:"longitude"`
	TextSearch          string `schema:"text_search"`
	PharmacyReference   string `schema:"reference"`
}

type PharmacyTextSearchResponse struct {
	Pharmacies []*pharmacy.PharmacyData `json:"results"`
}

func (p *PharmacyTextSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(PharmacyTextSearchRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "unable to parse input parameters: "+err.Error())
		return
	}

	patient, err := p.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		golog.Errorf("Unable to get patient information from auth token: " + err.Error())
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get the patient based on the auth token")
		return
	}

	var pharmacies []*pharmacy.PharmacyData
	if requestData.PharmacyReference != "" {
		pharmacyDetails, err := p.PharmacySearchService.GetPharmacyBasedOnId(requestData.PharmacyReference)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacy based on reference "+err.Error())
			return
		}
		pharmacies = []*pharmacy.PharmacyData{pharmacyDetails}
	} else {
		textSearch := "pharmacy " + requestData.TextSearch

		latitude := requestData.Latitude
		longitude := requestData.Longitude
		searchRadius := requestData.SearchRadiusInMiles
		if searchRadius == "" {
			searchRadius = defaultSearchRadiusInMeters
		}

		if requestData.Latitude == "" || requestData.Longitude == "" {
			// attempt to reverse geocode the zipcode if there is no specific location specified
			var locationInfo *maps.LocationInfo
			// attempt to reverse geocode the zipcode of the user
			if li, err := locationCache.Get(patient.ZipCode); err == nil && li != nil {
				locationInfo = li.(*maps.LocationInfo)
			} else {
				locationInfo, err = p.MapsService.GetLatLongFromSearchLocation(patient.ZipCode)
				if err == nil {
					locationCache.Set(patient.ZipCode, locationInfo)
				}
			}

			// fall back to including the zipcode in the text searchs
			// if we are unable to reverse geocode the zipcode
			if locationInfo == nil {
				textSearch = textSearch + " near " + patient.ZipCode
			} else {
				latitude = strconv.FormatFloat(locationInfo.Latitude, 'f', -1, 64)
				longitude = strconv.FormatFloat(locationInfo.Longitude, 'f', -1, 64)
			}
		}

		pharmacies, err = p.PharmacySearchService.GetPharmaciesBasedOnTextSearch(textSearch, latitude, longitude, searchRadius)
		for _, pharmacyData := range pharmacies {
			pharmacyData.Source = pharmacy.PHARMACY_SOURCE_GOOGLE
		}
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacies based on text search: "+err.Error())
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &PharmacyTextSearchResponse{Pharmacies: pharmacies})
}

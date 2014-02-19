package apiservice

import (
	"carefront/api"
	"carefront/libs/golog"
	"carefront/libs/maps"
	"carefront/libs/pharmacy"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/schema"
	"github.com/samuel/go-cache/cache"
)

var locationCache cache.Cache = cache.NewLFUCache(2048)

const (
	defaultSearchRadiusInMeters = "10000"
)

type PharmacyTextSearchHandler struct {
	PharmacySearchService pharmacy.PharmacySearchAPI
	MapsService           maps.MapsService
	DataApi               api.DataAPI
}

type PharmacyTextSearchRequestData struct {
	SearchRadiusInMeters string `schema:"search_radius_meters"`
	Latitude             string `schema:"latitude"`
	Longitude            string `schema:"longitude"`
	TextSearch           string `schema:"search_location"`
}

type PharmacyTextSearchResponse struct {
	Pharmacies []*pharmacy.PharmacyData `json:"pharmacy_results"`
}

func (p *PharmacyTextSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse form data: "+err.Error())
		return
	}

	var requestData PharmacyTextSearchRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
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

	// if there is no search string entered, default to searching for pharmacies around the zipcode of the patient
	textSearch := requestData.TextSearch
	if requestData.TextSearch == "" {
		textSearch = "pharmacy"
	}

	latitude := requestData.Latitude
	longitude := requestData.Longitude
	searchRadius := requestData.SearchRadiusInMeters
	if searchRadius == "" {
		searchRadius = defaultSearchRadiusInMeters
	}

	// Here's the algorithm for determining the area around which to search:
	// a) If the lat,lng is specified, search for pharmacies around the lat,lng with the specified or default search radius.
	// b) If not lat,lng specified, then reverse geocode the zipcode, and search for pharmacies around the zipcode wit the default search radius.
	// c) If the zipcode cannot be reverse geocoded because of issues with google geocoding service, then add the text "near [ZIPCODE]" to search for pharmacies around the zipcode,
	//    in this case the distance will not be specified.
	if requestData.Latitude == "" || requestData.Longitude == "" {
		// attempt to reverse geocode the zipcode if there is no specific location specified
		var locationInfo *maps.LocationInfo
		// attempt to reverse geocode the zipcode of the user
		if li, _ := locationCache.Get(patient.ZipCode); err == nil && li != nil {
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

	// if we got no pharmacies on the first try, perhaps its because the user entered just an address and not the name of a pharmacy.
	// in this case, lets try one more time by adding "pharmacy near [TEXT SEARCH ENTERED]" to see if we get any results
	if len(pharmacies) == 0 {
		// lets try the search again with the word pharmacy near in there
		pharmacies, err = p.PharmacySearchService.GetPharmaciesBasedOnTextSearch("pharmacy near "+textSearch, latitude, longitude, searchRadius)
	}

	// break down the results we get from google places api into the street, city and state
	breakdownAddressForPharmacies(pharmacies)

	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacies based on text search: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &PharmacyTextSearchResponse{Pharmacies: pharmacies})
}

func breakdownAddressForPharmacies(pharmacies []*pharmacy.PharmacyData) {
	for _, pharmacyData := range pharmacies {
		pharmacyData.Source = pharmacy.PHARMACY_SOURCE_GOOGLE
		addressComponents := strings.Split(pharmacyData.Address, ",")
		pharmacyData.Address = strings.TrimSpace(addressComponents[0])
		if len(addressComponents) > 1 {
			pharmacyData.City = strings.TrimSpace(addressComponents[1])
			if len(addressComponents) > 2 {
				pharmacyData.State = strings.TrimSpace(addressComponents[2])
			}
		}
	}
}

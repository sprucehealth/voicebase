package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/maps"
	"github.com/sprucehealth/backend/pharmacy"
)

const (
	defaultSearchRadiusInMeters = float64(10000)
	numResults                  = 50
)

type PharmacyTextSearchHandler struct {
	PharmacySearchService pharmacy.PharmacySearchAPI
	MapsService           maps.MapsService
	DataApi               api.DataAPI
}

type PharmacyTextSearchRequestData struct {
	SearchRadiusInMeters float64 `schema:"search_radius_meters"`
	Latitude             float64 `schema:"latitude"`
	Longitude            float64 `schema:"longitude"`
	TextSearch           string  `schema:"search_location"`
}

type PharmacyTextSearchResponse struct {
	Pharmacies []*pharmacy.PharmacyData `json:"pharmacy_results"`
}

func (p *PharmacyTextSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		http.NotFound(w, r)
		return
	}

	var requestData PharmacyTextSearchRequestData
	if err := DecodeRequestData(&requestData, r); err != nil {
		WriteValidationError(err.Error(), w, r)
		return
	}

	searchRadius := requestData.SearchRadiusInMeters
	if searchRadius == 0.0 {
		searchRadius = defaultSearchRadiusInMeters
	}

	pharmacies, err := p.PharmacySearchService.GetPharmaciesAroundSearchLocation(requestData.Latitude, requestData.Longitude, searchRadius, numResults)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	WriteJSON(w, &PharmacyTextSearchResponse{Pharmacies: pharmacies})
}

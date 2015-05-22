package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/pharmacy"
)

const (
	defaultSearchRadiusInMeters = float64(10000)
	numResults                  = 50
)

type pharmacySearchHandler struct {
	pharmacySearchAPI pharmacy.PharmacySearchAPI
	dataAPI           api.DataAPI
}

func NewPharmacySearchHandler(dataAPI api.DataAPI, pharmacySearchAPI pharmacy.PharmacySearchAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&pharmacySearchHandler{
			dataAPI:           dataAPI,
			pharmacySearchAPI: pharmacySearchAPI,
		}), httputil.Get)
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

func (p *pharmacySearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestData PharmacyTextSearchRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	searchRadius := requestData.SearchRadiusInMeters
	if searchRadius == 0.0 {
		searchRadius = defaultSearchRadiusInMeters
	}

	pharmacies, err := p.pharmacySearchAPI.GetPharmaciesAroundSearchLocation(requestData.Latitude, requestData.Longitude, searchRadius, numResults)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &PharmacyTextSearchResponse{Pharmacies: pharmacies})
}

package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/pharmacy"
)

type pharmacySearchHandler struct {
	dataAPI api.DataAPI
	erxAPI  erx.ERxAPI
}

func NewPharmacySearchHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI) http.Handler {
	return apiservice.AuthorizationRequired(&pharmacySearchHandler{
		dataAPI: dataAPI,
		erxAPI:  erxAPI,
	})
}

type PharmacySearchRequestData struct {
	ZipcodeString string   `schema:"zipcode_string"`
	PharmacyTypes []string `schema:"pharmacy_types[]"`
}

type PharmacySearchResponse struct {
	PharmacyResults []*pharmacy.PharmacyData `json:"pharmacy_results"`
}

func (d *pharmacySearchHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}
	return true, nil
}

func (d *pharmacySearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	requestData := &PharmacySearchRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	doctor, err := d.dataAPI.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	pharmacyResults, err := d.erxAPI.SearchForPharmacies(doctor.DoseSpotClinicianId, "", "", requestData.ZipcodeString, "", requestData.PharmacyTypes)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSON(w, &PharmacySearchResponse{PharmacyResults: pharmacyResults})
}

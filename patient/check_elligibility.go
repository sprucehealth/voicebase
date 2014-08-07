package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

type checkCareProvidingElligibilityHandler struct {
	dataAPI              api.DataAPI
	addressValidationAPI address.AddressValidationAPI
}

func NewCheckCareProvidingEligibilityHandler(dataAPI api.DataAPI, addressValidationAPI address.AddressValidationAPI) http.Handler {
	return &checkCareProvidingElligibilityHandler{
		dataAPI:              dataAPI,
		addressValidationAPI: addressValidationAPI,
	}
}

type CheckCareProvidingElligibilityRequestData struct {
	Zipcode string `schema:"zip_code,required"`
}

func (c *checkCareProvidingElligibilityHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}
	return true, nil
}

func (c *checkCareProvidingElligibilityHandler) NonAuthenticated() bool {
	return true
}

func (c *checkCareProvidingElligibilityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestData CheckCareProvidingElligibilityRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// given the zipcode, cover to city and state info
	cityStateInfo, err := c.addressValidationAPI.ZipcodeLookup(requestData.Zipcode)
	if err != nil {
		if err == address.InvalidZipcodeError {
			apiservice.WriteValidationError("Enter a valid zipcode", w, r)
			return
		}
		apiservice.WriteError(err, w, r)
		return
	}

	if cityStateInfo.StateAbbreviation == "" {
		apiservice.WriteValidationError("Enter valid zipcode", w, r)
		return
	}

	isAvailable, err := c.dataAPI.IsEligibleToServePatientsInState(cityStateInfo.StateAbbreviation, apiservice.HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, map[string]interface{}{
		"available":          isAvailable,
		"state":              cityStateInfo.State,
		"state_abbreviation": cityStateInfo.StateAbbreviation,
	})
}

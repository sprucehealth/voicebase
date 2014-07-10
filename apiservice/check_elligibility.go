package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
)

type CheckCareProvidingElligibilityHandler struct {
	DataApi              api.DataAPI
	AddressValidationApi address.AddressValidationAPI
	StaticContentUrl     string
}

type CheckCareProvidingElligibilityRequestData struct {
	Zipcode string `schema:"zip_code,required"`
}

func (c *CheckCareProvidingElligibilityHandler) NonAuthenticated() bool {
	return true
}

func (c *CheckCareProvidingElligibilityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		http.NotFound(w, r)
		return
	}

	var requestData CheckCareProvidingElligibilityRequestData
	if err := DecodeRequestData(&requestData, r); err != nil {
		WriteValidationError(err.Error(), w, r)
		return
	}

	// given the zipcode, cover to city and state info
	cityStateInfo, err := c.AddressValidationApi.ZipcodeLookup(requestData.Zipcode)
	if err != nil {
		if err == address.InvalidZipcodeError {
			WriteValidationError("Enter a valid zipcode", w, r)
			return
		}
		WriteError(err, w, r)
		return
	}

	if cityStateInfo.StateAbbreviation == "" {
		WriteValidationError("Enter valid zipcode", w, r)
		return
	}

	count, err := c.DataApi.EligibleCareProviderCountForState(cityStateInfo.StateAbbreviation, HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	// this means that there is no doctor available to server in the patient's entered
	// state for the condition
	if count == 0 {
		WriteError(NewNotEligibleToServePatientInStateError(), w, r)
		return
	}

	WriteJSON(w, map[string]interface{}{
		"state":              cityStateInfo.State,
		"state_abbreviation": cityStateInfo.StateAbbreviation,
	})
}

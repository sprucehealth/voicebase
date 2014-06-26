package apiservice

import (
	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"net/http"

	"github.com/sprucehealth/backend/third_party/github.com/gorilla/schema"
)

type CheckCareProvidingElligibilityHandler struct {
	DataApi              api.DataAPI
	AddressValidationApi address.AddressValidationAPI
	StaticContentUrl     string
}

type CheckCareProvidingElligibilityRequestData struct {
	Zipcode string `schema:"zip_code,required"`
}

type CheckCareProvidingElligibilityResponse struct {
	Doctor *common.Doctor `json:"doctor"`
}

func (c *CheckCareProvidingElligibilityHandler) NonAuthenticated() bool {
	return true
}

const (
	patientMessage = "We're not treating patients in your state yet."
)

func (c *CheckCareProvidingElligibilityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData CheckCareProvidingElligibilityRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input to check elligibility: "+err.Error())
		return
	}

	// given the zipcode, cover to city and state info
	cityStateInfo, err := c.AddressValidationApi.ZipcodeLookup(requestData.Zipcode)
	if err != nil {
		if err == address.InvalidZipcodeError {
			WriteUserError(w, http.StatusBadRequest, "Please enter a valid zipcode")
			return
		}

		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to use the maps service to reverse geocode the given zipcode to city and state information: "+err.Error())
		return
	}

	var doctorId int64
	if cityStateInfo.StateAbbreviation != "" {
		doctorId, err = c.DataApi.CheckCareProvidingElligibility(cityStateInfo.StateAbbreviation, HEALTH_CONDITION_ACNE_ID)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to check elligiblity for the patient to be seen by doctor: "+err.Error())
			return
		}
	}

	if doctorId != 0 {
		doctor, err := GetDoctorInfo(c.DataApi, doctorId, c.StaticContentUrl)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from id: "+err.Error())
			return
		}
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, &CheckCareProvidingElligibilityResponse{Doctor: doctor})
	} else {
		WriteUserError(w, http.StatusForbidden, patientMessage)
	}
}

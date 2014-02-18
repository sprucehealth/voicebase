package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/maps"
	"net/http"

	"github.com/gorilla/schema"
)

type CheckCareProvidingElligibilityHandler struct {
	DataApi          api.DataAPI
	MapsService      maps.MapsService
	StaticContentUrl string
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
	r.ParseForm()
	requestData := new(CheckCareProvidingElligibilityRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input to check elligibility: "+err.Error())
		return
	}

	// given the zipcode, cover to city and state info
	cityStateInfo, err := c.MapsService.ConvertZipcodeToCityState(requestData.Zipcode)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to use the maps service to reverse geocode the given zipcode to city and state information: "+err.Error())
		return
	}

	var doctorId int64
	if cityStateInfo != nil {
		doctorId, err = c.DataApi.CheckCareProvidingElligibility(cityStateInfo.ShortStateName, HEALTH_CONDITION_ACNE_ID)
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

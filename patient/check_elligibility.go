package patient

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type checkCareProvidingElligibilityHandler struct {
	dataAPI              api.DataAPI
	addressValidationAPI address.AddressValidationAPI
	analyticsLogger      analytics.Logger
}

func NewCheckCareProvidingEligibilityHandler(dataAPI api.DataAPI,
	addressValidationAPI address.AddressValidationAPI, analyticsLogger analytics.Logger) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&checkCareProvidingElligibilityHandler{
				dataAPI:              dataAPI,
				addressValidationAPI: addressValidationAPI,
				analyticsLogger:      analyticsLogger,
			}), []string{"GET"})
}

type CheckCareProvidingElligibilityRequestData struct {
	ZipCode   string `schema:"zip_code"`
	StateCode string `schema:"state_code"`
}

func (c *checkCareProvidingElligibilityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestData CheckCareProvidingElligibilityRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	var cityStateInfo *address.CityState
	var err error

	// resolve the provided zipcode to the state in the event that stateCode is not
	// already provided by the client
	if requestData.StateCode == "" {
		cityStateInfo, err = c.addressValidationAPI.ZipcodeLookup(requestData.ZipCode)
		if err == address.InvalidZipcodeError {
			apiservice.WriteValidationError("Enter a valid zipcode", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	} else {
		state, err := c.dataAPI.GetFullNameForState(requestData.StateCode)
		if api.IsErrNotFound(err) {
			apiservice.WriteValidationError("Enter valid state code", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		cityStateInfo = &address.CityState{
			State:             state,
			StateAbbreviation: requestData.StateCode,
		}
	}

	if cityStateInfo.StateAbbreviation == "" {
		apiservice.WriteValidationError("Enter valid zipcode or state code", w, r)
		return
	}

	// TODO: For now assume Acne pathway.
	pathway, err := c.dataAPI.PathwayForTag(api.AcnePathwayTag)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	isAvailable, err := c.dataAPI.IsEligibleToServePatientsInState(cityStateInfo.StateAbbreviation, pathway.ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	responseData := map[string]interface{}{
		"available":          isAvailable,
		"state":              cityStateInfo.State,
		"state_abbreviation": cityStateInfo.StateAbbreviation,
	}

	apiservice.WriteJSON(w, responseData)

	go func() {

		jsonData, err := json.Marshal(responseData)
		if err != nil {
			golog.Infof("Unable to marshal json: %s", err)
			return
		}
		c.analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "eligibility_check",
				Timestamp: analytics.Time(time.Now()),
				ExtraJSON: string(jsonData),
			},
		})
	}()
}

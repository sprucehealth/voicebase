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
	"golang.org/x/net/context"
)

type checkCareProvidingElligibilityHandler struct {
	dataAPI              api.DataAPI
	addressValidationAPI address.Validator
	analyticsLogger      analytics.Logger
}

func NewCheckCareProvidingEligibilityHandler(dataAPI api.DataAPI,
	addressValidationAPI address.Validator, analyticsLogger analytics.Logger) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&checkCareProvidingElligibilityHandler{
				dataAPI:              dataAPI,
				addressValidationAPI: addressValidationAPI,
				analyticsLogger:      analyticsLogger,
			}), httputil.Get)
}

type CheckCareProvidingElligibilityRequestData struct {
	ZipCode   string `schema:"zip_code"`
	StateCode string `schema:"state_code"`
}

func (c *checkCareProvidingElligibilityHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var requestData CheckCareProvidingElligibilityRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	var cityStateInfo *address.CityState
	var err error

	// resolve the provided zipcode to the state in the event that stateCode is not
	// already provided by the client
	if requestData.StateCode == "" {
		cityStateInfo, err = c.addressValidationAPI.ZipcodeLookup(requestData.ZipCode)
		if err == address.ErrInvalidZipcode {
			apiservice.WriteValidationError(ctx, "Enter a valid zipcode", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	} else {
		state, _, err := c.dataAPI.State(requestData.StateCode)
		if api.IsErrNotFound(err) {
			apiservice.WriteValidationError(ctx, "Enter valid state code", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		cityStateInfo = &address.CityState{
			State:             state,
			StateAbbreviation: requestData.StateCode,
		}
	}

	if cityStateInfo.StateAbbreviation == "" {
		apiservice.WriteValidationError(ctx, "Enter valid zipcode or state code", w, r)
		return
	}

	isAvailable, err := c.dataAPI.SpruceAvailableInState(cityStateInfo.StateAbbreviation)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	responseData := &struct {
		Available         bool   `json:"available"`
		State             string `json:"state"`
		StateAbbreviation string `json:"state_abbreviation"`
	}{
		Available:         isAvailable,
		State:             cityStateInfo.State,
		StateAbbreviation: cityStateInfo.StateAbbreviation,
	}
	httputil.JSONResponse(w, http.StatusOK, responseData)

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

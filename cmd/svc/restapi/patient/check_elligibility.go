package patient

import (
	"encoding/json"
	"net/http"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/address"
	"github.com/sprucehealth/backend/cmd/svc/restapi/analytics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/attribution"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type checkCareProvidingElligibilityHandler struct {
	dataAPI              api.DataAPI
	addressValidationAPI address.Validator
	analyticsLogger      analytics.Logger
}

// NewCheckCareProvidingEligibilityHandler returns and initialized instance of checkCareProvidingElligibilityHandler
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

// CheckCareProvidingElligibilityRequestData represents the data expected with a successful elligibility check request
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
		state, err := c.dataAPI.State(requestData.StateCode)
		if api.IsErrNotFound(err) {
			apiservice.WriteValidationError(ctx, "Enter valid state code", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		cityStateInfo = &address.CityState{
			State:             state.Name,
			StateAbbreviation: requestData.StateCode,
		}
	}

	if cityStateInfo.StateAbbreviation == "" {
		apiservice.WriteValidationError(ctx, "Enter valid zipcode or state code", w, r)
		return
	}

	// Check device attribution for information we may find relevant
	var careProviderID int64
	var isSprucePatient bool
	deviceID, err := apiservice.GetDeviceIDFromHeader(r)
	if err != nil {
		golog.Errorf("Couldn't get device ID from header for elligibility check: %s", err)
	} else {
		aData, err := c.dataAPI.LatestDeviceAttributionData(deviceID)
		if err != nil && !api.IsErrNotFound(err) {
			golog.Errorf("Couldn't get latest device attribution data for device ID: %s", deviceID)
		} else if !api.IsErrNotFound(err) {
			providerID, ok, err := aData.Int64Data(attribution.AKCareProviderID)
			if err != nil {
				golog.Errorf("Encountered error while checking for provider id in attribution data: %s", err)
			} else if ok {
				careProviderID = providerID
			}

			sp, ok, err := aData.BoolData(attribution.AKSprucePatient)
			if err != nil {
				golog.Errorf("Encountered error while checking for spruce patient flag: %s", err)
			} else if ok {
				isSprucePatient = sp
			}
		}
	}

	// If a provider ID is given then assume this is a practice extension case
	// so long as there is no flag to indicate we are dealing with spruce patient
	if careProviderID != 0 && !isSprucePatient {
		state, err := c.dataAPI.State(cityStateInfo.State)
		if err != nil {
			apiservice.WriteValidationError(ctx, "The state information is not valid", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		practiceModel, err := c.dataAPI.PracticeModel(careProviderID, state.ID)
		if err != nil && !api.IsErrNotFound(err) {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		if api.IsErrNotFound(err) || !practiceModel.HasPracticeExtension {
			apiservice.WriteValidationError(ctx, "The requested doctor is not available for practice extension in this location. Please email support@sprucehealth.com for assistance", w, r)
			return
		}
	}

	isAvailable := true
	if careProviderID == 0 {
		isAvailable, err = c.dataAPI.SpruceAvailableInState(cityStateInfo.StateAbbreviation)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
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

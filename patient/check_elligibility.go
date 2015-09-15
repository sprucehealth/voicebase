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
	// Note: This will transition to the attribution tracking in a upcoming change
	CareProviderID int64 `schema:"care_provider_id"`
}

func (c *checkCareProvidingElligibilityHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var requestData CheckCareProvidingElligibilityRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	var cityStateInfo *address.CityState
	var err error

	// If a provider ID is given then assumte this is a practice extension case
	if requestData.CareProviderID != 0 {
		doctor, err := c.dataAPI.GetDoctorFromID(requestData.CareProviderID)
		if api.IsErrNotFound(err) {
			apiservice.WriteValidationError(ctx, "The provided doctor ID is not valid", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		practiceModel, err := c.dataAPI.PracticeModel(doctor.ID.Int64())
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		if !practiceModel.HasPracticeExtension {
			apiservice.WriteValidationError(ctx, "The requested doctor is not available for practice extension", w, r)
			return
		}
	}

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

	isAvailable := true
	if requestData.CareProviderID == 0 {
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

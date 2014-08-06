package patient_case

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

type homeHandler struct {
	dataAPI              api.DataAPI
	authAPI              api.AuthAPI
	addressValidationAPI address.AddressValidationAPI
}

type homeResponse struct {
	Items []common.ClientView `json:"items"`
}

func NewHomeHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, addressValidationAPI address.AddressValidationAPI) http.Handler {
	return &homeHandler{
		dataAPI:              dataAPI,
		authAPI:              authAPI,
		addressValidationAPI: addressValidationAPI,
	}
}

// This handler needs to support both an authenticated
// and non-authentciated request so as to serve the appropriate home cards
// to the user in both cases
func (h *homeHandler) NonAuthenticated() bool {
	return true
}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// resolve zipcode to city and state information
	zipcode := r.FormValue("zip_code")
	if zipcode == "" {
		apiservice.WriteValidationError("zipcode required", w, r)
		return
	}

	var cityStateInfo *address.CityState
	cs, err := apiservice.ZipcodeCache.Get(zipcode)

	// get the info from the address validation service given that we are unable to look it up in the cache
	if err != nil || cs == nil {
		if err != nil {
			golog.Errorf("Unable to get zipcode from the cache: %s", err)
		}

		cityStateInfo, err = h.addressValidationAPI.ZipcodeLookup(zipcode)
		if err != nil {
			if err == address.InvalidZipcodeError {
				apiservice.WriteValidationError("Enter a valid zipcode", w, r)
				return
			}
			apiservice.WriteError(err, w, r)
			return
		}
	} else {
		cityStateInfo = cs.(*address.CityState)
	}

	// attempt to authenticate the user if the auth token is present
	authToken, err := apiservice.GetAuthTokenFromHeader(r)
	// if there is no auth header, handle the case of no account
	if err == apiservice.ErrNoAuthHeader {
		items, err := getHomeCards(nil, cityStateInfo, h.dataAPI)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		apiservice.WriteJSON(w, &homeResponse{Items: items})
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	account, err := h.authAPI.ValidateToken(authToken)
	if err != nil {
		apiservice.HandleAuthError(err, w)
		return
	}

	if account.Role != api.PATIENT_ROLE {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	patientId, err := h.dataAPI.GetPatientIdFromAccountId(account.ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientCases, err := h.dataAPI.GetCasesForPatient(patientId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var items []common.ClientView
	switch l := len(patientCases); {
	case l == 0:
		items, err = getHomeCards(nil, cityStateInfo, h.dataAPI)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	case l == 1:
		items, err = getHomeCards(patientCases[0], cityStateInfo, h.dataAPI)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	default:
		// FIX: Only supporting the case of 1 patient case for now given that we don't know how the home feed should
		// look when there are multiple cases
		apiservice.WriteError(fmt.Errorf("Expected only 1 patient case to exist instead got %d", len(patientCases)), w, r)
		return
	}

	apiservice.WriteJSON(w, &homeResponse{Items: items})
}

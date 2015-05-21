package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type homeHandler struct {
	dataAPI              api.DataAPI
	apiCDNDomain         string
	webDomain            string
	addressValidationAPI address.Validator
}

type homeResponse struct {
	ShowActionButton bool                `json:"show_action_button"`
	Items            []common.ClientView `json:"items"`
}

func NewHomeHandler(dataAPI api.DataAPI, apiCDNDomain, webDomain string, addressValidationAPI address.Validator) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&homeHandler{
			dataAPI:              dataAPI,
			apiCDNDomain:         apiCDNDomain,
			webDomain:            webDomain,
			addressValidationAPI: addressValidationAPI,
		}), []string{"GET"})
}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// use stateCode or resolve zipcode to city/state information
	zipcode := r.FormValue("zip_code")
	stateCode := r.FormValue("state_code")
	var cityStateInfo *address.CityState
	var err error
	if stateCode == "" {
		cityStateInfo, err = h.addressValidationAPI.ZipcodeLookup(zipcode)
		if err != nil {
			if err == address.ErrInvalidZipcode {
				apiservice.WriteValidationError("Enter a valid zipcode", w, r)
				return
			}
			apiservice.WriteError(err, w, r)
			return
		}
	} else {
		state, _, err := h.dataAPI.State(stateCode)
		if err != nil {
			apiservice.WriteValidationError("Enter valid state code", w, r)
			return
		}
		cityStateInfo = &address.CityState{
			State:             state,
			StateAbbreviation: stateCode,
		}
	}

	isSpruceAvailable, err := h.dataAPI.SpruceAvailableInState(cityStateInfo.State)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	ctxt := apiservice.GetContext(r)
	if ctxt.AccountID == 0 {
		items, err := getHomeCards(nil, cityStateInfo, isSpruceAvailable, h.dataAPI, h.apiCDNDomain, h.webDomain, r)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		httputil.JSONResponse(w, http.StatusOK, &homeResponse{
			ShowActionButton: isSpruceAvailable,
			Items:            items})
		return
	} else if ctxt.Role != api.RolePatient {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	patientID, err := h.dataAPI.GetPatientIDFromAccountID(ctxt.AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientCases, err := h.dataAPI.GetCasesForPatient(patientID, []string{
		common.PCStatusOpen.String(),
		common.PCStatusActive.String(),
		common.PCStatusInactive.String(),
		common.PCStatusPreSubmissionTriage.String()})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	items, err := getHomeCards(patientCases, cityStateInfo, isSpruceAvailable, h.dataAPI, h.apiCDNDomain, h.webDomain, r)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &homeResponse{
		ShowActionButton: isSpruceAvailable,
		Items:            items})
}

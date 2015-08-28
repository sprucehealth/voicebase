package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
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

func NewHomeHandler(dataAPI api.DataAPI, apiCDNDomain, webDomain string, addressValidationAPI address.Validator) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&homeHandler{
			dataAPI:              dataAPI,
			apiCDNDomain:         apiCDNDomain,
			webDomain:            webDomain,
			addressValidationAPI: addressValidationAPI,
		}), httputil.Get)
}

func (h *homeHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// use stateCode or resolve zipcode to city/state information
	zipcode := r.FormValue("zip_code")
	stateCode := r.FormValue("state_code")
	var cityStateInfo *address.CityState
	var err error
	if stateCode == "" {
		if zipcode == "" {
			apiservice.WriteValidationError(ctx, "Enter a valid zipcode or state", w, r)
			return
		}
		cityStateInfo, err = h.addressValidationAPI.ZipcodeLookup(zipcode)
		if err != nil {
			if err == address.ErrInvalidZipcode {
				apiservice.WriteValidationError(ctx, "Enter a valid zipcode", w, r)
				return
			}
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	} else {
		state, _, err := h.dataAPI.State(stateCode)
		if err != nil {
			apiservice.WriteValidationError(ctx, "Enter valid state code", w, r)
			return
		}
		cityStateInfo = &address.CityState{
			State:             state,
			StateAbbreviation: stateCode,
		}
	}

	isSpruceAvailable, err := h.dataAPI.SpruceAvailableInState(cityStateInfo.State)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	account, ok := apiservice.CtxAccount(ctx)
	if !ok {
		// Not authenticated

		items, err := getHomeCards(ctx, nil, nil, cityStateInfo, isSpruceAvailable, h.dataAPI, h.apiCDNDomain, h.webDomain, r)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		httputil.JSONResponse(w, http.StatusOK, &homeResponse{
			ShowActionButton: isSpruceAvailable,
			Items:            items})
		return
	}

	// Authenticated

	if account.Role != api.RolePatient {
		apiservice.WriteAccessNotAllowedError(ctx, w, r)
		return
	}

	patient, err := h.dataAPI.GetPatientFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	patientCases, err := h.dataAPI.GetCasesForPatient(patient.ID, []string{
		common.PCStatusOpen.String(),
		common.PCStatusActive.String(),
		common.PCStatusInactive.String(),
		common.PCStatusPreSubmissionTriage.String()})
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	items, err := getHomeCards(ctx, patient, patientCases, cityStateInfo, isSpruceAvailable, h.dataAPI, h.apiCDNDomain, h.webDomain, r)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &homeResponse{
		ShowActionButton: isSpruceAvailable && !patient.IsUnder18(),
		Items:            items})
}

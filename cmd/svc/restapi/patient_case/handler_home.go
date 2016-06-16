package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/address"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/feedback"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"golang.org/x/net/context"
)

type homeHandler struct {
	dataAPI              api.DataAPI
	feedbackClient       feedback.DAL
	apiCDNDomain         string
	webDomain            string
	addressValidationAPI address.Validator
}

type homeResponse struct {
	ShowActionButton bool                `json:"show_action_button"`
	Items            []common.ClientView `json:"items"`
}

func NewHomeHandler(dataAPI api.DataAPI, feedbackClient feedback.DAL, apiCDNDomain, webDomain string, addressValidationAPI address.Validator) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&homeHandler{
			dataAPI:              dataAPI,
			feedbackClient:       feedbackClient,
			apiCDNDomain:         apiCDNDomain,
			webDomain:            webDomain,
			addressValidationAPI: addressValidationAPI,
		}), httputil.Get, httputil.Delete)
}

func (h *homeHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		h.get(ctx, w, r)
	case httputil.Delete:
		h.delete(ctx, w, r)
	}
}

func (h *homeHandler) delete(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	if err := h.feedbackClient.UpdatePatientFeedback(id, &feedback.PatientFeedbackUpdate{
		Dismissed: ptr.Bool(true),
	}); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

func (h *homeHandler) get(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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
		state, err := h.dataAPI.State(stateCode)
		if err != nil {
			apiservice.WriteValidationError(ctx, "Enter valid state code", w, r)
			return
		}
		cityStateInfo = &address.CityState{
			State:             state.Name,
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

		items, err := getHomeCards(ctx, nil, nil, cityStateInfo, isSpruceAvailable, h.dataAPI, h.feedbackClient, h.apiCDNDomain, h.webDomain, r)
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

	items, err := getHomeCards(ctx, patient, patientCases, cityStateInfo, isSpruceAvailable, h.dataAPI, h.feedbackClient, h.apiCDNDomain, h.webDomain, r)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &homeResponse{
		ShowActionButton: isSpruceAvailable && !patient.IsUnder18(),
		Items:            items})
}

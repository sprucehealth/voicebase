package home

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type homeHandler struct {
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

type homeResponse struct {
	Items []PHView `json:"items"`
}

func NewHandler(dataAPI api.DataAPI, authAPI api.AuthAPI) *homeHandler {
	return &homeHandler{
		dataAPI: dataAPI,
		authAPI: authAPI,
	}
}

// This handler needs to support both an authenticated
// and non-authentciated request so as to serve the appropriate home cards
// to the user in both cases
func (h *homeHandler) NonAuthenticated() bool {
	return true
}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var account *common.Account

	// attempt to authenticate the user if the auth token is present
	authToken, err := apiservice.GetAuthTokenFromHeader(r)
	if err != apiservice.ErrNoAuthHeader && err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err != apiservice.ErrNoAuthHeader {
		account, err = h.authAPI.ValidateToken(authToken)
		if err != nil {
			apiservice.HandleAuthError(err, w)
			return
		}
	}

	if account == nil {
		items, err := getHomeCards(noAccountState, nil)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		apiservice.WriteJSON(w, &homeResponse{Items: items})
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

	var hState homeState
	if len(patientCases) == 0 {
		hState = noCaseState
	}

	items, err := getHomeCards(hState, nil)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &homeResponse{Items: items})
}

package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/auth"
	"github.com/sprucehealth/backend/common"
)

type authenticationHandler struct {
	authAPI             api.AuthAPI
	dataAPI             api.DataAPI
	smsAPI              api.SMSAPI
	fromNumber          string
	twoFactorExpiration int
}

type AuthenticationRequestData struct {
	Email    string `schema:"email,required"`
	Password string `schema:"password,required"`
}
type AuthenticationResponse struct {
	Token             string         `json:"token,omitempty"`
	Doctor            *common.Doctor `json:"doctor,omitempty"`
	LastFourPhone     string         `json:"last_four_phone,omitempty"`
	TwoFactorToken    string         `json:"two_factor_token,omitempty"`
	TwoFactorRequired bool           `json:"two_factor_required"`
}

func NewAuthenticationHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, smsAPI api.SMSAPI, fromNumber string, twoFactorExpiration int) http.Handler {
	return &authenticationHandler{
		dataAPI:             dataAPI,
		authAPI:             authAPI,
		smsAPI:              smsAPI,
		fromNumber:          fromNumber,
		twoFactorExpiration: twoFactorExpiration,
	}
}

func (d *authenticationHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}
	return true, nil
}

func (d *authenticationHandler) NonAuthenticated() bool {
	return true
}

func (d *authenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestData AuthenticationRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	account, err := d.authAPI.Authenticate(requestData.Email, requestData.Password)
	if err != nil {
		switch err {
		case api.LoginDoesNotExist, api.InvalidPassword:
			apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
			return
		}
		apiservice.WriteError(err, w, r)
		return
	}

	// Patient trying to sign in on doctor app
	if account.Role != api.DOCTOR_ROLE && account.Role != api.MA_ROLE {
		apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
		return
	}

	if account.TwoFactorEnabled {
		appHeaders := apiservice.ExtractSpruceHeaders(r)
		device, err := d.authAPI.GetAccountDevice(account.ID, appHeaders.DeviceID)
		if err != nil && err != api.NoRowsError {
			apiservice.WriteError(err, w, r)
			return
		}
		if device == nil || !device.Verified {
			// Create a temporary token to the client can use to authenticate the code submission request
			token, err := d.authAPI.CreateTempToken(account.ID, d.twoFactorExpiration, api.TwoFactorAuthToken, "")
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			phone, err := auth.SendTwoFactorCode(d.authAPI, d.smsAPI, d.fromNumber, account.ID, appHeaders.DeviceID, d.twoFactorExpiration)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			apiservice.WriteJSON(w, &AuthenticationResponse{
				LastFourPhone:     phone[len(phone)-4:],
				TwoFactorToken:    token,
				TwoFactorRequired: true,
			})
			return
		}
	}

	token, err := d.authAPI.CreateToken(account.ID, api.Mobile, api.RegularAuth)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctor, err := d.dataAPI.GetDoctorFromAccountId(account.ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &AuthenticationResponse{Token: token, Doctor: doctor})
}

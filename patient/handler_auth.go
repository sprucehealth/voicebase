// Package patient contains the AuthenticationHandler
//	Description:
//		Authenticate an existing user using their email and password
//
//	Request:
//		POST /v1/authenticate
//
//	Request-Body:
//		Content-Type: multipart/form-data
//		Parameters:
//			login=<username>
//			password=<password>
//
//	Response:
//		Content-Type: application/json
//		Content:
//			{
//				"token" : <auth_token>
//			}
// AuthenticationHandler is also responsible for signing up a new user
//	Description:
//		Sign up a new user with which to make authentication requests. As a result of signing up,
//		the user will get back an authorization token with which they can perform other tasks
//	 	that require authorization on the platform.
//
//	Request:
//		POST /v1/signup
//
//	Request-Body:
//		Content-Type: multipart/form-data
//		Parameters:
//			login=<email>
//			password=<password>
//
//	Response:
//		Content-Type: application/json
//		Content:
//			{
//				"token" : <auth_token>
//			}
// AuthenticationHandler is also responsible for logging out an existing user
//	Description:
//		Logout an existing, authorized user by invalidating the auth token such that it cannot be used
//		in future requests. The user will have to be re-authenticated to make any authorized requests
//		on the platform.
//
//	Request:
//		POST /v1/logout
//
//	Request-Headers:
//		{
//			"Authorization" : "token <auth_token>"
//		}
//
//	Response:
// 		Content-Type: text/plain
package patient

import (
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
)

type AuthenticationHandler struct {
	authApi              api.AuthAPI
	dataApi              api.DataAPI
	staticContentBaseUrl string
}

type AuthenticationResponse struct {
	Token   string          `json:"token"`
	Patient *common.Patient `json:"patient,omitempty"`
}

func NewAuthenticationHandler(dataApi api.DataAPI, authApi api.AuthAPI, staticContentBaseUrl string) *AuthenticationHandler {
	return &AuthenticationHandler{
		authApi:              authApi,
		dataApi:              dataApi,
		staticContentBaseUrl: staticContentBaseUrl,
	}
}

func (h *AuthenticationHandler) NonAuthenticated() bool {
	return true
}

type AuthRequestData struct {
	Login        string `schema:"login,required"`
	Password     string `schema:"password,required"`
	ExtendedAuth bool   `schema:"extended_auth"`
}

func (h *AuthenticationHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (h *AuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}
	action := strings.Split(r.URL.Path, "/")[2]
	// depending on whether we are signing up or logging in, make appropriate
	// call to service
	switch action {
	case "authenticate":
		var requestData AuthRequestData
		if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
			return
		}

		account, err := h.authApi.Authenticate(requestData.Login, requestData.Password, requestData.ExtendedAuth)
		if err != nil {
			switch err {
			case api.LoginDoesNotExist:
				golog.Context("AuthEvent", apiservice.AuthEventNoSuchLogin).Warningf(err.Error())
				apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
				return
			case api.InvalidPassword:
				golog.Context("AuthEvent", apiservice.AuthEventInvalidPassword).Warningf(err.Error())
				apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
				return
			default:
				apiservice.WriteError(err, w, r)
				return
			}
		}
		token, err := h.authApi.CreateToken(account.ID, api.Mobile)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		patient, err := h.dataApi.GetPatientFromAccountId(account.ID)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
			return
		}
		apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &AuthenticationResponse{Token: token, Patient: patient})
	case "logout":
		token, err := apiservice.GetAuthTokenFromHeader(r)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "authorization token not correctly specified in header")
			return
		}

		account, err := h.authApi.ValidateToken(token, api.Mobile)
		if err != nil {
			golog.Warningf("Unable to get account for token: %s", err)
		}

		if err := h.authApi.DeleteToken(token); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if account != nil {
			dispatch.Default.Publish(&AccountLoggedOutEvent{
				AccountId: account.ID,
			})
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

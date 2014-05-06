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
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/golog"
	"carefront/libs/pharmacy"
	thriftapi "carefront/thrift/api"
	"net/http"
	"strings"

	"github.com/gorilla/schema"
)

type AuthenticationHandler struct {
	authApi               thriftapi.Auth
	pharmacySearchService pharmacy.PharmacySearchAPI
	dataApi               api.DataAPI
	staticContentBaseUrl  string
}

type AuthenticationResponse struct {
	Token   string          `json:"token"`
	Patient *common.Patient `json:"patient,omitempty"`
	Doctor  *common.Doctor  `json:"doctor,omitempty"`
}

func NewAuthenticationHandler(dataApi api.DataAPI, authApi thriftapi.Auth, pharmacySearchService pharmacy.PharmacySearchAPI, staticContentBaseUrl string) *AuthenticationHandler {
	return &AuthenticationHandler{
		authApi:               authApi,
		pharmacySearchService: pharmacySearchService,
		dataApi:               dataApi,
		staticContentBaseUrl:  staticContentBaseUrl,
	}
}

func (h *AuthenticationHandler) NonAuthenticated() bool {
	return true
}

type AuthRequestData struct {
	Login    string `schema:"login,required"`
	Password string `schema:"password,required"`
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

		if res, err := h.authApi.LogIn(requestData.Login, requestData.Password); err != nil {
			switch err.(type) {
			case *thriftapi.NoSuchLogin:
				golog.Log("auth", golog.WARN, &apiservice.AuthLog{
					Event: apiservice.AuthEventNoSuchLogin,
				})
				apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
				return
			case *thriftapi.InvalidPassword:
				golog.Log("auth", golog.WARN, &apiservice.AuthLog{
					Event: apiservice.AuthEventInvalidPassword,
				})
				apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
				return
			default:
				// For now, treat all errors the same.
				apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
		} else {
			patient, err := h.dataApi.GetPatientFromAccountId(res.AccountId)
			if err != nil {
				apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
				return
			}
			doctor, err := apiservice.GetPrimaryDoctorInfoBasedOnPatient(h.dataApi, patient, h.staticContentBaseUrl)
			if err != nil {
				apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor based on patient: "+err.Error())
				return
			}
			apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &AuthenticationResponse{Token: res.Token, Patient: patient, Doctor: doctor})
		}
	case "logout":
		token, err := apiservice.GetAuthTokenFromHeader(r)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "authorization token not correctly specified in header")
			return
		}
		if err := h.authApi.LogOut(token); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// Package apiservice contains the AuthenticationHandler
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
package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/pharmacy"
	thriftapi "carefront/thrift/api"
	"github.com/gorilla/schema"
	"net/http"
	"strings"
)

type AuthenticationHandler struct {
	AuthApi               thriftapi.Auth
	PharmacySearchService pharmacy.PharmacySearchAPI
	DataApi               api.DataAPI
}

type AuthenticationResponse struct {
	Token   string          `json:"token"`
	Patient *common.Patient `json:"patient,omitempty"`
}

func (h *AuthenticationHandler) NonAuthenticated() bool {
	return true
}

type AuthRequestData struct {
	Login    string `schema:"login,required"`
	Password string `schema:"password,required"`
}

func (h *AuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	action := strings.Split(r.URL.Path, "/")[2]
	// depending on whether we are signing up or logging in, make appropriate
	// call to service
	switch action {
	case "authenticate":
		requestData := new(AuthRequestData)
		decoder := schema.NewDecoder()
		err := decoder.Decode(requestData, r.Form)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, err.Error())
			return
		}

		if res, err := h.AuthApi.Login(requestData.Login, requestData.Password); err != nil {
			switch err.(type) {
			case *thriftapi.NoSuchLogin:
				WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
				return
			default:
				// For now, treat all errors the same.
				WriteDeveloperError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
		} else {
			patient, err := GetPatientInfo(h.DataApi, h.PharmacySearchService, res.AccountId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
				return
			}
			WriteJSONToHTTPResponseWriter(w, http.StatusOK, &AuthenticationResponse{Token: res.Token, Patient: patient})
		}
	case "isauthenticated":
		token, err := GetAuthTokenFromHeader(r)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "authoriation token not correctly specified in the header: "+err.Error())
			return
		}
		validTokenResponse, err := h.AuthApi.ValidateToken(token)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "unable to validate auth token: "+err.Error())
			return
		}

		if validTokenResponse.IsValid == false {
			WriteAuthTimeoutError(w)
			return
		}
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
	case "logout":
		token, err := GetAuthTokenFromHeader(r)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "authorization token not correctly specified in header")
			return
		}
		if err := h.AuthApi.Logout(token); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

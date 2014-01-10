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
	"log"
	"net/http"
	"strings"

	"carefront/api"
	"carefront/common"
	thriftapi "carefront/thrift/api"
	"github.com/gorilla/schema"
)

type AuthenticationHandler struct {
	AuthApi thriftapi.Auth
	DataApi api.DataAPI
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
	case "signup":
		requestData := new(AuthRequestData)
		decoder := schema.NewDecoder()
		if err := decoder.Decode(requestData, r.Form); err != nil {
			log.Printf("apiservice/auth: failed to parse request data: %+v", err)
			WriteDeveloperError(w, http.StatusBadRequest, err.Error())
			return
		}

		if res, err := h.AuthApi.Signup(requestData.Login, requestData.Password); err != nil {
			switch err.(type) {
			case *thriftapi.LoginAlreadyExists:
				WriteUserError(w, http.StatusBadRequest, "Login already exists")
				return
			default:
				log.Printf("apiservice/auth: Signup RPC call failed: %+v", err)
				// For now, treat all errors the same.
				WriteDeveloperError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
		} else {
			patient, err := h.DataApi.GetPatientFromAccountId(res.AccountId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from account id:  "+err.Error())
				return
			}
			WriteJSONToHTTPResponseWriter(w, http.StatusOK, &AuthenticationResponse{Token: res.Token, Patient: patient})
		}
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
			patient, err := h.DataApi.GetPatientFromAccountId(res.AccountId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from account id:  "+err.Error())
				return
			}
			WriteJSONToHTTPResponseWriter(w, http.StatusOK, &AuthenticationResponse{Token: res.Token, Patient: patient})
		}
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

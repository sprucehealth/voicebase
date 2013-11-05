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
	"errors"
	"log"
	"net/http"
	"strings"
)

type AuthenticationHandler struct {
	AuthApi api.Auth
}

type AuthenticationResponse struct {
	Token string `json:"token"`
}

type AuthenticationErrorResponse struct {
	ErrorString string `json:"error"`
}

func (h *AuthenticationHandler) NonAuthenticated() bool {
	return true
}

func (h *AuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	action := strings.Split(r.URL.String(), "/")[2]
	// depending on whether we are signing up or logging in, make appropriate
	// call to service
	switch action {
	case "signup":
		login, password, err := getLoginAndPassword(r)
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if token, _, err := h.AuthApi.Signup(login, password); err == api.ErrSignupFailedUserExists {
			w.WriteHeader(http.StatusBadRequest)
			WriteJSONToHTTPResponseWriter(w, AuthenticationErrorResponse{err.Error()})
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			WriteJSONToHTTPResponseWriter(w, AuthenticationErrorResponse{err.Error()})
		} else {
			if err := WriteJSONToHTTPResponseWriter(w, AuthenticationResponse{token}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}

		}
	case "authenticate":
		login, password, err := getLoginAndPassword(r)
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if token, _, err := h.AuthApi.Login(login, password); err == api.ErrLoginFailed {
			w.WriteHeader(http.StatusForbidden)
		} else if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if err := WriteJSONToHTTPResponseWriter(w, AuthenticationResponse{token}); err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

		}
	case "logout":
		token, err := GetAuthTokenFromHeader(r)
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		err = h.AuthApi.Logout(token)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func getLoginAndPassword(r *http.Request) (login, password string, err error) {
	login = r.FormValue("login")
	password = r.FormValue("password")
	if login == "" || password == "" {
		return "", "", errors.New("login and or password missing!")
	}
	return login, password, nil
}

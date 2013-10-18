package main

import (
	"carefront/api"
	"encoding/json"
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
			enc := json.NewEncoder(w)
			enc.Encode(AuthenticationErrorResponse{err.Error()})
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			enc := json.NewEncoder(w)
			enc.Encode(AuthenticationErrorResponse{err.Error()})
		} else {
			enc := json.NewEncoder(w)
			if err := enc.Encode(AuthenticationResponse{token}); err != nil {
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
			enc := json.NewEncoder(w)
			if err := enc.Encode(AuthenticationResponse{token}); err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	case "logout":
		token, err := GetAuthTokenFromHeader(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = h.AuthApi.Logout(token)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
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

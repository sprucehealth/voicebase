package main

import (
	"encoding/json"
	"log"
	"net/http"
	"carefront/api"
	"fmt"
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
	login := r.FormValue("login")
	password := r.FormValue("password")
	if login == "" || password == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	action := strings.Split(r.URL.String(), "/")[2]
	fmt.Println(action)
	// depending on whether we are signing up or logging in, make appropriate 
	// call to service
	if action == "signup" {
		if token, err :=  h.AuthApi.Signup(login, password);  err == api.ErrSignupFailedUserExists {
			w.WriteHeader(http.StatusBadRequest)
			enc :=	json.NewEncoder(w)
			enc.Encode(AuthenticationErrorResponse{err.Error()})
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			enc := json.NewEncoder(w)
			enc.Encode(AuthenticationErrorResponse{err.Error()})
		} else {
			enc := json.NewEncoder(w)
			if err:= enc.Encode(AuthenticationResponse{token}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	} else {
		if token, err := h.AuthApi.Login(login, password); err == api.ErrLoginFailed {
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
	}
}

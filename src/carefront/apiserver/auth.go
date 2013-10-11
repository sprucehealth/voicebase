package main

import (
	"encoding/json"
	"log"
	"net/http"

	"carefront/api"
)

type AuthenticationHandler struct {
	AuthApi api.Auth
}

type AuthenticationResponse struct {
	Token string `json:"token"`
}

func (h *AuthenticationHandler) NonAuthenticated() bool {
	return true
}

func (h *AuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	login := r.FormValue("login")
	password := r.FormValue("password")
	if login == "" || password == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
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

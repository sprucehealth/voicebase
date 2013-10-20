package main

import (
	"carefront/api"
	"log"
	"net/http"
)

// If a handler conforms to this interface and returns true then
// non-authenticated requests will be handled. Otherwise,
// they 403 response will be returned.
type NonAuthenticated interface {
	NonAuthenticated() bool
}

type AuthServeMux struct {
	http.ServeMux
	AuthApi api.Auth
}

// Parse the "Authorization: token xxx" header and check the token for validity
func (mux *AuthServeMux) checkAuth(r *http.Request) (bool, error) {
	token, err := GetAuthTokenFromHeader(r)
	if err == ErrBadAuthToken {
		return false, nil
	} else if err != nil {
		return false, err
	}
	valid, _, err := mux.AuthApi.ValidateToken(token)
	return valid, err
}

func (mux *AuthServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
	if r.RequestURI == "*" {
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h, _ := mux.Handler(r)
	if nonAuth, ok := h.(NonAuthenticated); !ok || !nonAuth.NonAuthenticated() {
		if valid, err := mux.checkAuth(r); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if !valid {
			w.WriteHeader(http.StatusForbidden)
			return
		}

	}
	h.ServeHTTP(w, r)
}

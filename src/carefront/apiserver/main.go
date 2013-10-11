package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"carefront/api"
)

// If a handler conforms to this protocol and returns true then
// non-authenticated requests will be handled. Otherwise,
// they will be given a 403 response.
type NonAuthenticated interface {
	NonAuthenticated() bool
}

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

type PingHandler int

func (h PingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("pong\n")); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

type AuthServeMux struct {
	http.ServeMux
	AuthApi api.Auth
}

func (mux *AuthServeMux) checkAuth(r *http.Request) (bool, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false, nil
	}
	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "token" {
		return false, nil
	}
	valid, _, err := mux.AuthApi.ValidateToken(parts[1])
	return valid, err
}

func (mux *AuthServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func main() {
	authApi := &api.MockAuth{
		Accounts: map[string]api.MockAccount{
			"fu": api.MockAccount{
				Id:       1,
				Password: "bar",
			},
		},
	}

	mux := &AuthServeMux{*http.NewServeMux(), authApi}

	authHandler := &AuthenticationHandler{authApi}
	pingHandler := PingHandler(0)
	mux.Handle("/v1/authenticate", authHandler)
	mux.Handle("/v1/ping", pingHandler)

	s := &http.Server{
		Addr:    ":8080",
		Handler: mux,
		// ReadTimeout:    10 * time.Second,
		// WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}

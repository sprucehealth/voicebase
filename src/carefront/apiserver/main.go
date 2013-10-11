package main

import (
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	"carefront/api"
)

var (
	flagListenAddr = flag.String("listen", ":8080", "Address and port to listen on")
)

// If a handler conforms to this interface and returns true then
// non-authenticated requests will be handled. Otherwise,
// they 403 response will be returned.
type NonAuthenticated interface {
	NonAuthenticated() bool
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

// Parse the "Authorization: token xxx" header and check the token for validity
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
	flag.Parse()

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
		Addr:           *flagListenAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}

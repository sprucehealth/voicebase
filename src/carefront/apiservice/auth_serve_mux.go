package apiservice

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

type CustomResponseWriter struct {
	WrappedResponseWriter http.ResponseWriter
	StatusCode            int
	WroteHeader           bool
}

func (c *CustomResponseWriter) WriteHeader(status int) {
	c.StatusCode = status
	c.WroteHeader = true
	c.WrappedResponseWriter.WriteHeader(status)
}

func (c *CustomResponseWriter) Header() http.Header {
	return c.WrappedResponseWriter.Header()
}

func (c *CustomResponseWriter) Write(bytes []byte) (int, error) {
	if c.WroteHeader == false {
		c.WriteHeader(http.StatusOK)
	}
	return (c.WrappedResponseWriter.Write(bytes))
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
	customResponseWriter := &CustomResponseWriter{w, 0, false}
	defer func() { log.Printf("%s %s %s %d\n", r.RemoteAddr, r.Method, r.URL, customResponseWriter.StatusCode) }()
	if r.RequestURI == "*" {
		customResponseWriter.Header().Set("Connection", "close")
		customResponseWriter.WriteHeader(http.StatusBadRequest)
		return
	}
	h, _ := mux.Handler(r)
	if nonAuth, ok := h.(NonAuthenticated); !ok || !nonAuth.NonAuthenticated() {
		if valid, err := mux.checkAuth(r); err != nil {
			log.Println(err)
			customResponseWriter.WriteHeader(http.StatusInternalServerError)
			return
		} else if !valid {
			customResponseWriter.WriteHeader(http.StatusForbidden)
			return
		}

	}
	h.ServeHTTP(customResponseWriter, r)
}

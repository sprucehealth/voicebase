package apiservice

import (
	"log"
	"net/http"

	"carefront/thrift/api"
	"github.com/samuel/go-metrics/metrics"
)

// If a handler conforms to this interface and returns true then
// non-authenticated requests will be handled. Otherwise,
// they 403 response will be returned.
type NonAuthenticated interface {
	NonAuthenticated() bool
}

type Authenticated interface {
	AccountIdFromAuthToken(accountId int64)
}

type AuthServeMux struct {
	http.ServeMux
	AuthApi api.Auth

	statRequests    metrics.Counter
	statAuthSuccess metrics.Counter
	statAuthFailure metrics.Counter
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

func NewAuthServeMux(authApi api.Auth, statsRegistry metrics.Registry) *AuthServeMux {
	mux := &AuthServeMux{
		ServeMux:        *http.NewServeMux(),
		AuthApi:         authApi,
		statRequests:    metrics.NewCounter(),
		statAuthSuccess: metrics.NewCounter(),
		statAuthFailure: metrics.NewCounter(),
	}
	statsRegistry.Add("requests/total", mux.statRequests)
	statsRegistry.Add("requests/auth/success", mux.statAuthSuccess)
	statsRegistry.Add("requests/auth/failure", mux.statAuthFailure)
	return mux
}

// Parse the "Authorization: token xxx" header and check the token for validity
func (mux *AuthServeMux) checkAuth(r *http.Request) (bool, int64, error) {
	token, err := GetAuthTokenFromHeader(r)
	if err == ErrBadAuthToken {
		return false, 0, nil
	} else if err != nil {
		return false, 0, err
	}
	if res, err := mux.AuthApi.ValidateToken(token); err != nil {
		return false, 0, err
	} else {
		var accountId int64
		if res.AccountId != nil {
			accountId = *res.AccountId
		}
		return res.IsValid, accountId, nil
	}
}

func (mux *AuthServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.statRequests.Inc(1)

	customResponseWriter := &CustomResponseWriter{w, 0, false}
	defer func() {
		DeleteContext(r)
		log.Printf("%s %s %s %d %s\n", r.RemoteAddr, r.Method, r.URL, customResponseWriter.StatusCode, w.Header().Get("Content-Type"))
	}()
	if r.RequestURI == "*" {
		customResponseWriter.Header().Set("Connection", "close")
		customResponseWriter.WriteHeader(http.StatusBadRequest)
		return
	}
	h, pattern := mux.Handler(r)

	// these means the page is not found, in which case serve the page as we would
	// since we have a page not found handler returned
	if pattern == "" {
		h.ServeHTTP(customResponseWriter, r)
		return
	}

	if nonAuth, ok := h.(NonAuthenticated); !ok || !nonAuth.NonAuthenticated() {
		if valid, accountId, err := mux.checkAuth(r); err != nil {
			log.Println(err)
			customResponseWriter.WriteHeader(http.StatusInternalServerError)
			return
		} else if !valid {
			mux.statAuthFailure.Inc(1)
			WriteAuthTimeoutError(customResponseWriter)
			return
		} else {
			mux.statAuthSuccess.Inc(1)
			ctx := GetContext(r)
			ctx.AccountId = accountId
			if auth, ok := h.(Authenticated); ok {
				auth.AccountIdFromAuthToken(accountId)
			}
		}
	}
	h.ServeHTTP(customResponseWriter, r)
}

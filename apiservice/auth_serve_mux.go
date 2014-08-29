package apiservice

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

// If a handler conforms to this interface and returns true then
// non-authenticated requests will be handled. Otherwise,
// they 403 response will be returned.
type NonAuthenticated interface {
	NonAuthenticated() bool
}

// Authorized interface helps ensure that caller of every handler is authorized
// to process the call it is intended for. Every handler in the restapi must implement this interface
type Authorized interface {
	IsAuthorized(r *http.Request) (bool, error)
}

type AuthServeMux struct {
	http.ServeMux
	authApi         api.AuthAPI
	analyticsLogger analytics.Logger

	statLatency              metrics.Histogram
	statRequests             metrics.Counter
	statResponseCodeRequests map[int]metrics.Counter
	statAuthSuccess          metrics.Counter
	statAuthFailure          metrics.Counter
	statIDGenFailure         metrics.Counter
	statIDGenSuccess         metrics.Counter
}

type AuthEvent string

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

const (
	AuthEventNoSuchLogin     AuthEvent = "NoSuchLogin"
	AuthEventInvalidPassword AuthEvent = "InvalidPassword"
	AuthEventInvalidToken    AuthEvent = "InvalidToken"
)

func NewAuthServeMux(authApi api.AuthAPI, analyticsLogger analytics.Logger, statsRegistry metrics.Registry) *AuthServeMux {
	mux := &AuthServeMux{
		ServeMux:         *http.NewServeMux(),
		authApi:          authApi,
		analyticsLogger:  analyticsLogger,
		statLatency:      metrics.NewBiasedHistogram(),
		statRequests:     metrics.NewCounter(),
		statAuthSuccess:  metrics.NewCounter(),
		statAuthFailure:  metrics.NewCounter(),
		statIDGenFailure: metrics.NewCounter(),
		statIDGenSuccess: metrics.NewCounter(),
		statResponseCodeRequests: map[int]metrics.Counter{
			http.StatusOK:                  metrics.NewCounter(),
			http.StatusForbidden:           metrics.NewCounter(),
			http.StatusNotFound:            metrics.NewCounter(),
			http.StatusInternalServerError: metrics.NewCounter(),
			http.StatusBadRequest:          metrics.NewCounter(),
			http.StatusMethodNotAllowed:    metrics.NewCounter(),
		},
	}
	statsRegistry.Add("requests/latency", mux.statLatency)
	statsRegistry.Add("requests/total", mux.statRequests)
	statsRegistry.Add("requests/auth/success", mux.statAuthSuccess)
	statsRegistry.Add("requests/auth/failure", mux.statAuthFailure)
	statsRegistry.Add("requests/idgen/failure", mux.statIDGenFailure)
	statsRegistry.Add("requests/idgen/success", mux.statIDGenSuccess)

	for statusCode, counter := range mux.statResponseCodeRequests {
		statsRegistry.Add(fmt.Sprintf("requests/response/%d", statusCode), counter)
	}

	return mux
}

// Parse the "Authorization: token xxx" header and check the token for validity
func (mux *AuthServeMux) checkAuth(r *http.Request) (*common.Account, error) {
	if Testing {
		if idStr := r.Header.Get("AccountID"); idStr != "" {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				return nil, err
			}
			return mux.authApi.GetAccount(id)
		}
	}

	token, err := GetAuthTokenFromHeader(r)
	if err != nil {
		return nil, err
	}
	return mux.authApi.ValidateToken(token, api.Mobile)
}

func (mux *AuthServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.statRequests.Inc(1)

	ctx := GetContext(r)
	ctx.RequestStartTime = time.Now()
	var err error
	ctx.RequestID, err = idgen.NewID()
	if err != nil {
		golog.Errorf("Unable to generate a requestId: %s", err)
		mux.statIDGenFailure.Inc(1)
	} else {
		mux.statIDGenSuccess.Inc(1)
	}

	customResponseWriter := &CustomResponseWriter{w, 0, false}

	// Use strict transport security. Not entirely useful for a REST API, but it doesn't hurt.
	// http://en.wikipedia.org/wiki/HTTP_Strict_Transport_Security
	customResponseWriter.Header().Set("Strict-Transport-Security", "max-age=31536000")

	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]

			reqID := GetContext(r).RequestID
			remoteAddr := r.RemoteAddr
			if idx := strings.LastIndex(remoteAddr, ":"); idx > 0 {
				remoteAddr = remoteAddr[:idx]
			}

			golog.Context(
				"StatusCode", 500,
				"RequestID", reqID,
				"RemoteAddr", remoteAddr,
				"Method", r.Method,
				"URL", r.URL.String(),
				"UserAgent", r.UserAgent(),
			).Criticalf("http: panic: %v\n%s", err, buf)

			mux.analyticsLogger.WriteEvents([]analytics.Event{
				&analytics.WebRequestEvent{
					Service:      "restapi",
					Path:         r.URL.Path,
					Timestamp:    analytics.Time(time.Now()),
					RequestID:    reqID,
					StatusCode:   500,
					Method:       r.Method,
					URL:          r.URL.String(),
					RemoteAddr:   remoteAddr,
					ContentType:  w.Header().Get("Content-Type"),
					UserAgent:    r.UserAgent(),
					ResponseTime: int(time.Since(ctx.RequestStartTime).Nanoseconds() / 1e3),
				},
			})

			if !customResponseWriter.WroteHeader {
				w.WriteHeader(http.StatusInternalServerError)
			}
			mux.statResponseCodeRequests[http.StatusInternalServerError].Inc(1)
		} else {
			responseTime := time.Since(ctx.RequestStartTime).Nanoseconds() / 1e3
			mux.statLatency.Update(responseTime)

			remoteAddr := r.RemoteAddr
			if idx := strings.LastIndex(remoteAddr, ":"); idx > 0 {
				remoteAddr = remoteAddr[:idx]
			}

			statusCode := customResponseWriter.StatusCode
			if statusCode == 0 {
				statusCode = 200
			}
			if counter, ok := mux.statResponseCodeRequests[statusCode]; ok {
				counter.Inc(1)
			}
			reqID := GetContext(r).RequestID
			golog.Context(
				"StatusCode", statusCode,
				"Method", r.Method,
				"URL", r.URL.String(),
				"RequestID", reqID,
				"RemoteAddr", remoteAddr,
				"ContentType", w.Header().Get("Content-Type"),
				"UserAgent", r.UserAgent(),
				"ResponseTime", float64(responseTime)/1000.0,
			).LogDepthf(-1, golog.INFO, "apirequest")

			mux.analyticsLogger.WriteEvents([]analytics.Event{
				&analytics.WebRequestEvent{
					Service:      "restapi",
					Path:         r.URL.Path,
					Timestamp:    analytics.Time(time.Now()),
					RequestID:    reqID,
					StatusCode:   statusCode,
					Method:       r.Method,
					URL:          r.URL.String(),
					RemoteAddr:   remoteAddr,
					ContentType:  w.Header().Get("Content-Type"),
					UserAgent:    r.UserAgent(),
					ResponseTime: int(responseTime),
				},
			})
		}
		DeleteContext(r)
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
		account, err := mux.checkAuth(r)
		if err == nil {

			mux.statAuthSuccess.Inc(1)
			ctx.AccountId = account.ID
			ctx.Role = account.Role
		} else {
			mux.statAuthFailure.Inc(1)
			HandleAuthError(err, customResponseWriter, r)
			return
		}
	}

	// ensure that every handler is authorized to carry out its call
	if isAuthorized, err := h.(Authorized).IsAuthorized(r); err != nil {
		WriteError(err, customResponseWriter, r)
		return
	} else if !isAuthorized {
		WriteAccessNotAllowedError(customResponseWriter, r)
		return
	}

	h.ServeHTTP(customResponseWriter, r)
}

// Package patient contains the AuthenticationHandler
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
package patient

import (
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/auth"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ratelimit"
)

type AuthenticationHandler struct {
	authAPI              api.AuthAPI
	dataAPI              api.DataAPI
	dispatcher           *dispatch.Dispatcher
	staticContentBaseURL string
	rateLimiter          ratelimit.KeyedRateLimiter
	statLoginAttempted   *metrics.Counter
	statLoginSucceeded   *metrics.Counter
	statLoginRateLimited *metrics.Counter
}

type AuthenticationResponse struct {
	Token   string          `json:"token"`
	Patient *common.Patient `json:"patient,omitempty"`
}

func NewAuthenticationHandler(
	dataAPI api.DataAPI, authAPI api.AuthAPI, dispatcher *dispatch.Dispatcher,
	staticContentBaseURL string, rateLimiter ratelimit.KeyedRateLimiter,
	metricsRegistry metrics.Registry,
) http.Handler {
	h := &AuthenticationHandler{
		authAPI:              authAPI,
		dataAPI:              dataAPI,
		dispatcher:           dispatcher,
		staticContentBaseURL: staticContentBaseURL,
		rateLimiter:          rateLimiter,
		statLoginAttempted:   metrics.NewCounter(),
		statLoginSucceeded:   metrics.NewCounter(),
		statLoginRateLimited: metrics.NewCounter(),
	}
	metricsRegistry.Add("login.attempted", h.statLoginAttempted)
	metricsRegistry.Add("login.succeeded", h.statLoginSucceeded)
	metricsRegistry.Add("login.rate-limited", h.statLoginRateLimited)
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(h),
		[]string{"POST"})
}

type AuthRequestData struct {
	Login        string `schema:"login,required"`
	Password     string `schema:"password,required"`
	ExtendedAuth bool   `schema:"extended_auth"`
}

func (h *AuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}
	action := strings.Split(r.URL.Path, "/")[2]
	// depending on whether we are signing up or logging in, make appropriate
	// call to service
	switch action {
	case "authenticate":
		h.statLoginAttempted.Inc(1)

		// rate limit on IP address (prevent scanning accounts)
		if ok, err := h.rateLimiter.Check("login:"+r.RemoteAddr, 1); err != nil {
			golog.Errorf("Rate limit check failed: %s", err.Error())
		} else if !ok {
			h.statLoginRateLimited.Inc(1)
			apiservice.WriteAccessNotAllowedError(w, r)
			return
		}

		var requestData AuthRequestData
		if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
			return
		}

		requestData.Login = strings.TrimSpace(strings.ToLower(requestData.Login))

		// rate limit on account (prevent trying one account from multiple IPs)
		if ok, err := h.rateLimiter.Check("login:"+requestData.Login, 1); err != nil {
			golog.Errorf("Rate limit check failed: %s", err.Error())
		} else if !ok {
			h.statLoginRateLimited.Inc(1)
			apiservice.WriteAccessNotAllowedError(w, r)
			return
		}

		account, err := h.authAPI.Authenticate(requestData.Login, requestData.Password)
		if err != nil {
			switch err {
			case api.ErrLoginDoesNotExist:
				golog.Context("AuthEvent", apiservice.AuthEventNoSuchLogin).Warningf(err.Error())
				apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
				return
			case api.ErrInvalidPassword:
				golog.Context("AuthEvent", apiservice.AuthEventInvalidPassword).Warningf(err.Error())
				apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
				return
			default:
				apiservice.WriteError(err, w, r)
				return
			}
		}
		token, err := h.authAPI.CreateToken(account.ID, api.Mobile, requestData.ExtendedAuth)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		patient, err := h.dataAPI.GetPatientFromAccountID(account.ID)
		if api.IsErrNotFound(err) {
			golog.Warningf("Non-patient sign in attempt at patient endpoint (account %d)", account.ID)
			apiservice.WriteUserError(w, http.StatusForbidden, "Invalid email/password combination")
			return
		} else if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
			return
		}

		headers := apiservice.ExtractSpruceHeaders(r)
		h.dispatcher.PublishAsync(&auth.AuthenticatedEvent{
			AccountID:     patient.AccountID.Int64(),
			SpruceHeaders: headers,
		})

		h.statLoginSucceeded.Inc(1)

		httputil.JSONResponse(w, http.StatusOK, &AuthenticationResponse{Token: token, Patient: patient})
	case "logout":
		token, err := apiservice.GetAuthTokenFromHeader(r)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "authorization token not correctly specified in header")
			return
		}

		account, err := h.authAPI.ValidateToken(token, api.Mobile)
		if err != nil {
			golog.Warningf("Unable to get account for token: %s", err)
		}

		if err := h.authAPI.DeleteToken(token); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if account != nil {
			h.dispatcher.Publish(&AccountLoggedOutEvent{
				AccountID: account.ID,
			})
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

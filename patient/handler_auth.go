package patient

import (
	"net/http"
	"strings"

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

const actionNeededSimpleFeedbackPrompt = "simple_feedback_prompt"

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

type ActionNeeded struct {
	Type string `json:"type"`
}

type AuthenticationResponse struct {
	Token         string          `json:"token"`
	Patient       *common.Patient `json:"patient,omitempty"`
	ActionsNeeded []*ActionNeeded `json:"actions_needed,omitempty"`
}

type AuthRequestData struct {
	Login        string `schema:"login,required" json:"login"`
	Password     string `schema:"password,required" json:"password"`
	ExtendedAuth bool   `schema:"extended_auth" json:"extended_auth"`
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
		httputil.Post)
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
		h.authenticate(w, r)
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

func (h *AuthenticationHandler) authenticate(w http.ResponseWriter, r *http.Request) {
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
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
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
	var ctOpt api.CreateTokenOption
	if requestData.ExtendedAuth {
		ctOpt |= api.CreateTokenExtended
	}
	token, err := h.authAPI.CreateToken(account.ID, api.Mobile, ctOpt)
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

	res := &AuthenticationResponse{
		Token:   token,
		Patient: patient,
	}
	if showFeedback(h.dataAPI, patient.PatientID.Int64()) {
		res.ActionsNeeded = append(res.ActionsNeeded, &ActionNeeded{Type: actionNeededSimpleFeedbackPrompt})
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

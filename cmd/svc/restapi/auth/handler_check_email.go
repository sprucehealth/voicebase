package auth

import (
	"net/http"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/validate"
)

type EmailChecker interface {
	AccountForEmail(email string) (*common.Account, error)
}

type checkEmailHandler struct {
	emailChecker    EmailChecker
	metricsRegistry metrics.Registry
	statRequests    *metrics.Counter
	statAvailable   *metrics.Counter
	statUnavailable *metrics.Counter
}

type emailCheckResponse struct {
	Available bool `json:"available"`
}

func NewCheckEmailHandler(emailChecker EmailChecker, rateLimiter ratelimit.KeyedRateLimiter, metricsRegistry metrics.Registry) http.Handler {
	h := &checkEmailHandler{
		emailChecker:    emailChecker,
		metricsRegistry: metricsRegistry,
		statRequests:    metrics.NewCounter(),
		statAvailable:   metrics.NewCounter(),
		statUnavailable: metrics.NewCounter(),
	}
	metricsRegistry.Add("requests", h.statRequests)
	metricsRegistry.Add("available", h.statAvailable)
	metricsRegistry.Add("unavailable", h.statUnavailable)
	return apiservice.NoAuthorizationRequired(httputil.SupportedMethods(
		ratelimit.RemoteAddrHandler(h, rateLimiter, "check_email:", metricsRegistry),
		httputil.Get))
}

func (h *checkEmailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	em := r.FormValue("email")
	if em == "" {
		// Don't record this in the stats since it's basically a noop
		httputil.JSONResponse(w, http.StatusOK, emailCheckResponse{Available: false})
		return
	}

	h.statRequests.Inc(1)

	if !validate.Email(em) {
		apiservice.WriteUserError(w, http.StatusBadRequest, "The provided email address is invalid.")
		return
	}

	_, err := h.emailChecker.AccountForEmail(em)
	if err == api.ErrLoginDoesNotExist {
		h.statAvailable.Inc(1)
		httputil.JSONResponse(w, http.StatusOK, emailCheckResponse{Available: true})
	} else if err != nil {
		apiservice.WriteError(err, w, r)
	} else {
		h.statUnavailable.Inc(1)
		httputil.JSONResponse(w, http.StatusOK, emailCheckResponse{Available: false})
	}
}

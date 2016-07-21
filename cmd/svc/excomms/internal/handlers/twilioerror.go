package handlers

import (
	"net/http"

	"github.com/samuel/go-metrics/metrics"
)

type twilioErrorHandler struct {
	statTwilioError *metrics.Counter
}

func NewTwilioErrorHandler(statTwilioError *metrics.Counter) http.Handler {
	return &twilioErrorHandler{
		statTwilioError: statTwilioError,
	}
}

func (t *twilioErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.statTwilioError.Inc(1)
	w.WriteHeader(http.StatusOK)
}

package router

import (
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
)

type routeMetricsHandler struct {
	H            http.Handler
	statRequests *metrics.Counter
}

func (h routeMetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Increment the counter representing requests to this API
	h.statRequests.Inc(1)
	h.H.ServeHTTP(w, r)
}

func NewRouteMetricsHandler(h http.Handler, path string, metricsRegistry metrics.Registry) http.Handler {
	// Scope this metric to the URI of the API
	scope := metricsRegistry.Scope(strings.Replace(strings.ToLower(path), `/`, `.`, -1))
	handler := routeMetricsHandler{
		H:            h,
		statRequests: metrics.NewCounter(),
	}
	scope.Add(`requests`, handler.statRequests)
	return handler
}

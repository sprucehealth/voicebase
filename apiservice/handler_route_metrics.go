package apiservice

import (
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
)

type routeMetricsHandler struct {
	H            http.Handler
	statRequests *metrics.Counter
	statLatency  metrics.Histogram
}

func (h routeMetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := GetContext(r)

	// Defer execution to post latency metrics
	defer func() {
		responseTime := time.Since(ctx.RequestStartTime).Nanoseconds() / 1e3
		h.statLatency.Update(responseTime)
	}()

	// Increment the counter representing requests to this API
	h.statRequests.Inc(1)
	h.H.ServeHTTP(w, r)
}

func NewRouteMetricsHandler(h http.Handler, path string, metricsRegistry metrics.Registry) http.Handler {
	// Scope this metric to the URI of the API
	scope := metricsRegistry.Scope(`restapi` + strings.ToLower(path))
	handler := routeMetricsHandler{
		H:            h,
		statRequests: metrics.NewCounter(),
		statLatency:  metrics.NewBiasedHistogram(),
	}
	scope.Add(`requests`, handler.statRequests)
	scope.Add(`latency`, handler.statLatency)
	return handler
}

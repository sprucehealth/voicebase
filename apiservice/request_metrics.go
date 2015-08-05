package apiservice

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
)

var hostname string

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown"
		golog.Errorf("Failed to get hostname: %s", err.Error())
	}
}

type AuthEvent string

const (
	AuthEventNoSuchLogin     AuthEvent = "NoSuchLogin"
	AuthEventInvalidPassword AuthEvent = "InvalidPassword"
	AuthEventInvalidToken    AuthEvent = "InvalidToken"
)

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

type routeMetricSet struct {
	Requests *metrics.Counter
	Latency  metrics.Histogram
}

type metricsHandler struct {
	h                        QueryableMux
	dispatcher               *dispatch.Dispatcher
	statLatency              metrics.Histogram
	statRequests             *metrics.Counter
	statResponseCodeRequests map[int]*metrics.Counter
	statAuthSuccess          *metrics.Counter
	statAuthFailure          *metrics.Counter
	statIDGenFailure         *metrics.Counter
	statIDGenSuccess         *metrics.Counter
	routeMetricSets          map[string]*routeMetricSet
}

func MetricsHandler(h QueryableMux, dispatcher *dispatch.Dispatcher, statsRegistry metrics.Registry) http.Handler {
	m := &metricsHandler{
		h:                h,
		dispatcher:       dispatcher,
		statLatency:      metrics.NewBiasedHistogram(),
		statRequests:     metrics.NewCounter(),
		statAuthSuccess:  metrics.NewCounter(),
		statAuthFailure:  metrics.NewCounter(),
		statIDGenFailure: metrics.NewCounter(),
		statIDGenSuccess: metrics.NewCounter(),
		statResponseCodeRequests: map[int]*metrics.Counter{
			http.StatusOK:                  metrics.NewCounter(),
			http.StatusForbidden:           metrics.NewCounter(),
			http.StatusNotFound:            metrics.NewCounter(),
			http.StatusInternalServerError: metrics.NewCounter(),
			http.StatusBadRequest:          metrics.NewCounter(),
			http.StatusMethodNotAllowed:    metrics.NewCounter(),
		},
		routeMetricSets: make(map[string]*routeMetricSet),
	}

	statsRegistry.Add("requests/latency", m.statLatency)
	statsRegistry.Add("requests/total", m.statRequests)
	statsRegistry.Add("requests/auth/success", m.statAuthSuccess)
	statsRegistry.Add("requests/auth/failure", m.statAuthFailure)
	statsRegistry.Add("requests/idgen/failure", m.statIDGenFailure)
	statsRegistry.Add("requests/idgen/success", m.statIDGenSuccess)
	for statusCode, counter := range m.statResponseCodeRequests {
		statsRegistry.Add(fmt.Sprintf("requests/response/%d", statusCode), counter)
	}
	for _, path := range h.SupportedPaths() {
		metricSet := &routeMetricSet{
			Requests: metrics.NewCounter(),
			Latency:  metrics.NewBiasedHistogram(),
		}
		m.routeMetricSets[path] = metricSet
		scope := statsRegistry.Scope(strings.ToLower(path)[1:]) // 1: to remove the first slash
		scope.Add(`requests`, metricSet.Requests)
		scope.Add(`latency`, metricSet.Latency)
	}

	return m
}

func (h *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.statRequests.Inc(1)

	ctx := GetContext(r)
	ctx.RequestStartTime = time.Now()
	var err error
	ctx.RequestID, err = idgen.NewID()
	if err != nil {
		golog.Errorf("Unable to generate a requestId: %s", err)
		h.statIDGenFailure.Inc(1)
	} else {
		h.statIDGenSuccess.Inc(1)
	}

	customResponseWriter := &CustomResponseWriter{w, 0, false}

	// Use strict transport security. Not entirely useful for a REST API, but it doesn't hurt.
	// http://en.wikipedia.org/wiki/HTTP_Strict_Transport_Security
	customResponseWriter.Header().Set("Strict-Transport-Security", "max-age=31536000")

	// write the requestID to the response header so that we have a way to track
	// back to a particular request for which the response was generated
	customResponseWriter.Header().Set("S-Request-ID", strconv.FormatUint(ctx.RequestID, 10))

	defer func() {
		err := recover()

		DeleteContext(r)

		responseTime := time.Since(ctx.RequestStartTime).Nanoseconds() / 1e3

		statusCode := customResponseWriter.StatusCode
		if statusCode == 0 {
			statusCode = 200
		}

		remoteAddr := r.RemoteAddr
		if idx := strings.LastIndex(remoteAddr, ":"); idx > 0 {
			remoteAddr = remoteAddr[:idx]
		}

		if err != nil {
			statusCode = 500

			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]

			golog.Context(
				"StatusCode", statusCode,
				"RequestID", ctx.RequestID,
				"AccountID", ctx.AccountID,
				"RemoteAddr", remoteAddr,
				"Method", r.Method,
				"URL", r.URL.String(),
				"UserAgent", r.UserAgent(),
			).Criticalf("http: panic: %v\n%s", err, buf)

			if !customResponseWriter.WroteHeader {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			// FIXME: This is a bit of a hack to ignore media uploads in the
			// performance metrics. Since we don't track this per path it's
			// more useful to ignore this since it adds too much noise.
			if r.URL.Path != "/v1/media" {
				h.statLatency.Update(responseTime)
			}

			if !environment.IsTest() {
				golog.Context(
					"StatusCode", statusCode,
					"Method", r.Method,
					"URL", r.URL.String(),
					"RequestID", ctx.RequestID,
					"AccountID", ctx.AccountID,
					"RemoteAddr", remoteAddr,
					"ContentType", w.Header().Get("Content-Type"),
					"UserAgent", r.UserAgent(),
					"ResponseTime", float64(responseTime)/1000.0,
				).LogDepthf(-1, golog.INFO, "apirequest")
			}
		}

		if counter, ok := h.statResponseCodeRequests[statusCode]; ok {
			counter.Inc(1)
		}

		headers := ExtractSpruceHeaders(r)
		h.dispatcher.Publish(
			&analytics.WebRequestEvent{
				Service:      "restapi",
				Path:         r.URL.Path,
				Timestamp:    analytics.Time(ctx.RequestStartTime),
				RequestID:    ctx.RequestID,
				StatusCode:   statusCode,
				Method:       r.Method,
				URL:          r.URL.String(),
				RemoteAddr:   remoteAddr,
				ContentType:  w.Header().Get("Content-Type"),
				UserAgent:    r.UserAgent(),
				ResponseTime: int(responseTime),
				Server:       hostname,
				AccountID:    ctx.AccountID,
				DeviceID:     headers.DeviceID,
			},
		)
	}()

	if r.RequestURI == "*" {
		customResponseWriter.Header().Set("Connection", "close")
		customResponseWriter.WriteHeader(http.StatusBadRequest)
		return
	}

	if h.h.IsSupportedPath(r.URL.Path) {
		h.beginRouteMetric(r)
		defer func() {
			h.endRouteMetric(r)
		}()
	}

	h.h.ServeHTTP(customResponseWriter, r)
}

func (h *metricsHandler) beginRouteMetric(r *http.Request) {
	metricSet, ok := h.routeMetricSets[r.URL.Path]
	if !ok {
		golog.Errorf("Unable to begin route metrics for path %v - it was never opened", r.URL.Path)
		return
	}
	metricSet.Requests.Inc(1)
}

func (h *metricsHandler) endRouteMetric(r *http.Request) {
	ctx := GetContext(r)
	responseTime := time.Since(ctx.RequestStartTime).Nanoseconds() / 1e3
	metricSet, ok := h.routeMetricSets[r.URL.Path]
	if !ok {
		golog.Errorf("Unable to end route metrics for path %v - it was never opened", r.URL.Path)
		return
	}
	metricSet.Latency.Update(responseTime)
}

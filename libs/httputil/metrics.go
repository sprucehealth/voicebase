package httputil

import (
	"fmt"
	"net/http"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"golang.org/x/net/context"
)

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *metricsResponseWriter) WriteHeader(status int) {
	w.statusCode = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *metricsResponseWriter) Write(bytes []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(bytes)
}

type metricsHandler struct {
	h                   ContextHandler
	statLatency         metrics.Histogram
	statResponseCodeMap map[int]*metrics.Counter
}

// MetricsHandler wraps a handler to provides stats counters on response codes.
func MetricsHandler(h ContextHandler, metricsRegistry metrics.Registry) ContextHandler {
	m := &metricsHandler{
		h:           h,
		statLatency: metrics.NewBiasedHistogram(),
		statResponseCodeMap: map[int]*metrics.Counter{
			http.StatusOK:                  metrics.NewCounter(),
			http.StatusBadRequest:          metrics.NewCounter(),
			http.StatusInternalServerError: metrics.NewCounter(),
			http.StatusForbidden:           metrics.NewCounter(),
			http.StatusMethodNotAllowed:    metrics.NewCounter(),
			http.StatusNotFound:            metrics.NewCounter(),
		},
	}

	for statusCode, counter := range m.statResponseCodeMap {
		metricsRegistry.Add(fmt.Sprintf("requests/response/%d", statusCode), counter)
	}
	metricsRegistry.Add("requests/latency", m.statLatency)

	return m
}

func (m *metricsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	metricsrw := &metricsResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
	startTime := time.Now()

	defer func() {
		m.statLatency.Update(int64(time.Since(startTime) / 1e3))
		if err := recover(); err != nil {
			m.statResponseCodeMap[http.StatusInternalServerError].Inc(1)
			if !metricsrw.wroteHeader {
				metricsrw.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			if counter, ok := m.statResponseCodeMap[metricsrw.statusCode]; ok {
				counter.Inc(1)
			}
		}
	}()

	m.h.ServeHTTP(ctx, metricsrw, r)
}

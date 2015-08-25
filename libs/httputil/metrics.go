package httputil

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"golang.org/x/net/context"
)

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode       int
	wroteHeader      bool
	firstByteWritten uint32
	firstByteTime    time.Time
}

func (w *metricsResponseWriter) WriteHeader(status int) {
	w.statusCode = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *metricsResponseWriter) Write(bytes []byte) (int, error) {
	// Record the first time Write is called to get latency to start of response.
	// This is useful for handlers that might take a while to return the reponse such
	// as large media to slow clients.
	if atomic.SwapUint32(&w.firstByteWritten, 1) == 0 {
		w.firstByteTime = time.Now()
	}
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(bytes)
}

type metricsHandler struct {
	h                    ContextHandler
	statLatency          metrics.Histogram
	statFirstByteLatency metrics.Histogram
	statRequests         *metrics.Counter
	statResponseCodeMap  map[int]*metrics.Counter
}

// MetricsHandler wraps a handler to provides stats counters on response codes.
func MetricsHandler(h ContextHandler, metricsRegistry metrics.Registry) ContextHandler {
	m := &metricsHandler{
		h:                    h,
		statLatency:          metrics.NewBiasedHistogram(),
		statFirstByteLatency: metrics.NewBiasedHistogram(),
		statRequests:         metrics.NewCounter(),
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
	metricsRegistry.Add("requests/total", m.statRequests)
	metricsRegistry.Add("requests/latency", m.statLatency)
	metricsRegistry.Add("requests/first-byte-latency", m.statFirstByteLatency)

	return m
}

func (m *metricsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	m.statRequests.Inc(1)

	metricsrw := &metricsResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
	startTime := time.Now()

	defer func() {
		now := time.Now()
		dt := now.Sub(startTime)
		if metricsrw.firstByteTime.IsZero() {
			// If Write was never called on the response then use the total response time instead
			m.statFirstByteLatency.Update(int64(dt / 1e3))
		} else {
			m.statFirstByteLatency.Update(int64(now.Sub(metricsrw.firstByteTime) / 1e3))
		}
		m.statLatency.Update(int64(dt / 1e3))
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

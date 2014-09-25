package httputil

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
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
	statResponseCodeMap map[int]metrics.Counter
	h                   http.Handler
}

func MetricsHandler(h http.Handler, metricsRegistry metrics.Registry) http.Handler {
	m := &metricsHandler{
		h: h,
		statResponseCodeMap: map[int]metrics.Counter{
			http.StatusOK:                  metrics.NewCounter(),
			http.StatusBadRequest:          metrics.NewCounter(),
			http.StatusInternalServerError: metrics.NewCounter(),
			http.StatusForbidden:           metrics.NewCounter(),
			http.StatusMethodNotAllowed:    metrics.NewCounter(),
		},
	}

	for statusCode, counter := range m.statResponseCodeMap {
		metricsRegistry.Add(fmt.Sprintf("requests/response/%d", statusCode), counter)
	}

	return m
}

func (m *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metricsrw := &metricsResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	defer func() {
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

	m.h.ServeHTTP(metricsrw, r)
}

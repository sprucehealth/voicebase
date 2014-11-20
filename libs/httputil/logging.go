package httputil

/*
FIXME: This package uses the analytics package which is unfortunate
because it's tightly coupled. Ideally a better solution should be found
that doesn't require this relationship to exist.
*/

import (
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/analytics"
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

type ContextKey int

const (
	requestIDContextKey ContextKey = iota
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.statusCode = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(bytes []byte) (int, error) {
	if w.wroteHeader == false {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(bytes)
}

func RequestID(r *http.Request) int64 {
	reqID, _ := context.Get(r, requestIDContextKey).(int64)
	return reqID
}

type requestIDHandler struct {
	h http.Handler
}

func RequestIDHandler(h http.Handler) http.Handler {
	return &requestIDHandler{h: h}
}

func (h *requestIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID, err := idgen.NewID()
	if err != nil {
		requestID = 0
		golog.Errorf("Failed to generate request ID: %s", err.Error())
	}
	context.Set(r, requestIDContextKey, requestID)
	h.h.ServeHTTP(w, r)
}

type loggingHandler struct {
	h    http.Handler
	log  golog.Logger
	alog analytics.Logger
}

func LoggingHandler(h http.Handler, log golog.Logger, alog analytics.Logger) http.Handler {
	return &loggingHandler{
		h:    h,
		log:  log,
		alog: alog,
	}
}

func (h *loggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logrw := &loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// http://en.wikipedia.org/wiki/HTTP_Strict_Transport_Security
	logrw.Header().Set("Strict-Transport-Security", "max-age=31536000")

	startTime := time.Now()

	// Save the URL here incase it gets mangled by the time
	// the defer gets called. This can happen when suing http.StripPrefix
	// such as for static file serving.
	url := r.URL.String()
	defer func() {
		rerr := recover()

		responseTime := time.Since(startTime).Nanoseconds() / 1e3

		reqID := RequestID(r)
		remoteAddr := r.RemoteAddr
		if idx := strings.LastIndex(remoteAddr, ":"); idx > 0 {
			remoteAddr = remoteAddr[:idx]
		}
		log := h.log.Context(
			"Method", r.Method,
			"URL", url,
			"UserAgent", r.UserAgent(),
			"RequestID", reqID,
			"RemoteAddr", remoteAddr,
		)
		statusCode := logrw.statusCode
		if rerr != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Context(
				"StatusCode", http.StatusInternalServerError,
			).Criticalf("http: panic: %v\n%s", rerr, buf)
			if !logrw.wroteHeader {
				w.WriteHeader(http.StatusInternalServerError)
			}
			statusCode = http.StatusInternalServerError
		} else {
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			log.Context(
				"StatusCode", statusCode,
			).LogDepthf(-1, golog.INFO, "webrequest")
		}

		h.alog.WriteEvents([]analytics.Event{
			&analytics.WebRequestEvent{
				Service:      "www", // TODO: hardcoded for now
				Path:         r.URL.Path,
				Timestamp:    analytics.Time(startTime),
				RequestID:    reqID,
				StatusCode:   statusCode,
				Method:       r.Method,
				URL:          r.URL.String(),
				RemoteAddr:   remoteAddr,
				ContentType:  w.Header().Get("Content-Type"),
				UserAgent:    r.UserAgent(),
				ResponseTime: int(responseTime),
				Server:       hostname,
			},
		})
	}()

	h.h.ServeHTTP(logrw, r)
}

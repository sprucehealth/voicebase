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

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/environment"
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

// contextKey provides a unique type to be used as a private namespace for values in the context.
type contextKey int

const (
	requestIDContextKey contextKey = iota
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

// RequestID returns the request ID for an HTTP request. RequestIDHandler
// must be used to guarantee that a request ID exists. If a request ID does
// not exist because a handler has not been wrapped with RequestIDHandler then
// this function will panic.
func RequestID(ctx context.Context) int64 {
	reqID, _ := ctx.Value(requestIDContextKey).(int64)
	return reqID
}

type requestIDHandler struct {
	h ContextHandler
}

// RequestIDHandler wraps a handler to provide generation of a unique
// request ID per request. The ID is available by calling RequestID(request).
func RequestIDHandler(h ContextHandler) ContextHandler {
	return &requestIDHandler{h: h}
}

func (h *requestIDHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestID, err := idgen.NewID()
	if err != nil {
		requestID = 0
		golog.Errorf("Failed to generate request ID: %s", err.Error())
	}
	h.h.ServeHTTP(context.WithValue(ctx, requestIDContextKey, requestID), w, r)
}

type loggingHandler struct {
	h    ContextHandler
	log  golog.Logger
	alog analytics.Logger
}

// LoggingHandler wraps a handler to provide request logging.
func LoggingHandler(h ContextHandler, log golog.Logger, alog analytics.Logger) ContextHandler {
	if environment.IsTest() {
		return h
	}
	return &loggingHandler{
		h:    h,
		log:  log,
		alog: alog,
	}
}

func (h *loggingHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

		reqID := RequestID(ctx)
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
				Referrer:     r.Referer(),
			},
		})
	}()

	h.h.ServeHTTP(ctx, logrw, r)
}

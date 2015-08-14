package httputil

/*
FIXME: This package uses the analytics package which is unfortunate
because it's tightly coupled. Ideally a better solution should be found
that doesn't require this relationship to exist.
*/

import (
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"golang.org/x/net/context"
)

var requestEventPool = sync.Pool{
	New: func() interface{} {
		return &RequestEvent{}
	},
}

var loggingResponseWriterPool = sync.Pool{
	New: func() interface{} {
		return &loggingResponseWriter{}
	},
}

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

// RequestEvent is a request/response log event
type RequestEvent struct {
	Timestamp       time.Time
	ResponseTime    time.Duration
	ServerHostname  string
	StatusCode      int
	ResponseHeaders http.Header
	Request         *http.Request
	// URL is provided separate from the request as it was copied before calling sub
	// handlers as they might change the URL (e.g. http.StripPrefix)
	URL *url.URL
	// RemoteAddr is a normalized version of r.RemoteAddr that removes any port number
	RemoteAddr string
	// Panic amd StackTrace are set if a sub handler panics
	Panic      interface{}
	StackTrace []byte
}

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
// this returns 0.
func RequestID(ctx context.Context) uint64 {
	reqID, _ := ctx.Value(requestIDContextKey).(uint64)
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
	w.Header().Set("S-Request-ID", strconv.FormatUint(requestID, 10))
	h.h.ServeHTTP(context.WithValue(ctx, requestIDContextKey, requestID), w, r)
}

// LogFunc is a function that logs http request events. The RequestEvent object is only
// valid during the call and should not be kept after it returns.
type LogFunc func(context.Context, *RequestEvent)

type loggingHandler struct {
	h    ContextHandler
	alog LogFunc
}

// LoggingHandler wraps a handler to provide request logging.
func LoggingHandler(h ContextHandler, alog LogFunc) ContextHandler {
	return &loggingHandler{
		h:    h,
		alog: alog,
	}
}

func (h *loggingHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	logrw := loggingResponseWriterPool.Get().(*loggingResponseWriter)
	*logrw = loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	startTime := time.Now()

	// Save the URL here incase it gets mangled by the time
	// the defer gets called. This can happen when using http.StripPrefix
	// such as for static file serving.
	// url := r.URL.String()
	// path := r.URL.Path
	earl := *r.URL
	defer func() {
		rerr := recover()

		remoteAddr := r.RemoteAddr
		if idx := strings.LastIndex(remoteAddr, ":"); idx > 0 {
			remoteAddr = remoteAddr[:idx]
		}
		ev := requestEventPool.Get().(*RequestEvent)
		*ev = RequestEvent{
			Timestamp:       startTime,
			StatusCode:      logrw.statusCode,
			ResponseHeaders: logrw.Header(),
			Request:         r,
			URL:             &earl,
			RemoteAddr:      remoteAddr,
			ResponseTime:    time.Since(startTime),
			ServerHostname:  hostname,
		}
		if rerr != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			ev.Panic = rerr
			ev.StackTrace = buf
			if !logrw.wroteHeader {
				w.WriteHeader(http.StatusInternalServerError)
			}
			ev.StatusCode = http.StatusInternalServerError
		} else {
			if ev.StatusCode == 0 {
				ev.StatusCode = http.StatusOK
			}
		}

		h.alog(ctx, ev)

		requestEventPool.Put(ev)
		loggingResponseWriterPool.Put(logrw)
	}()

	h.h.ServeHTTP(ctx, logrw, r)
}

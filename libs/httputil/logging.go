package httputil

import (
	"context"
	"encoding/base64"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

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

type requestIDContextKey struct{}

// LogMapContextKey is used for referencing a map
// in the context object to be used as key/value storage
// to log contextual information.
type logMapContextKey struct{}

// CtxLogMap returns access to the log map that can be used to add
// contextual information for logging purposes. If the logMap
// doesn't exist, then returns false.
func CtxLogMap(ctx context.Context) *conc.Map {
	m, _ := ctx.Value(logMapContextKey{}).(*conc.Map)
	return m
}

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
// this returns an empty string.
func RequestID(ctx context.Context) string {
	reqID, _ := ctx.Value(requestIDContextKey{}).(string)
	return reqID
}

// CtxWithRequestID adds a request ID to the context
func CtxWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, id)
}

type requestIDHandler struct {
	h http.Handler
}

// RequestIDHandler wraps a handler to provide generation of a unique
// request ID per request. The ID is available by calling RequestID(request).
func RequestIDHandler(h http.Handler) http.Handler {
	return &requestIDHandler{h: h}
}

func (h *requestIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := newRequestID()
	if err != nil {
		golog.Errorf("Failed to generate new request ID: %s", err)
	}
	logger := golog.ContextLogger(r.Context()).Context("request_id", id)
	r = r.WithContext(golog.WithLogger(r.Context(), logger))
	h.h.ServeHTTP(w, r.WithContext(CtxWithRequestID(r.Context(), id)))
}

func newRequestID() (string, error) {
	var b [16]byte
	n, err := rnd.Read(b[:])
	if err != nil {
		return "", errors.Trace(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:n]), nil
}

// LogFunc is a function that logs http request events. The RequestEvent object is only
// valid during the call and should not be kept after it returns.
type LogFunc func(context.Context, *RequestEvent)

type loggingHandler struct {
	h           http.Handler
	appName     string
	behindProxy bool
	alog        LogFunc
}

// LoggingHandler wraps a handler to provide request logging. alog is optional, but
// if provided it overrides the default logging to golog.
func LoggingHandler(h http.Handler, appName string, behindProxy bool, alog LogFunc) http.Handler {
	return &loggingHandler{
		h:           h,
		behindProxy: behindProxy,
		appName:     appName,
		alog:        alog,
	}
}

func (h *loggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logrw := loggingResponseWriterPool.Get().(*loggingResponseWriter)
	*logrw = loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	startTime := time.Now()

	ctx := r.Context()
	ctx = context.WithValue(ctx, logMapContextKey{}, conc.NewMap())

	// Save the URL here incase it gets mangled by the time
	// the defer gets called. This can happen when using http.StripPrefix
	// such as for static file serving.
	// url := r.URL.String()
	// path := r.URL.Path
	earl := *r.URL
	defer func() {
		rerr := recover()

		ev := requestEventPool.Get().(*RequestEvent)
		*ev = RequestEvent{
			Timestamp:       startTime,
			StatusCode:      logrw.statusCode,
			ResponseHeaders: logrw.Header(),
			Request:         r,
			URL:             &earl,
			RemoteAddr:      RemoteAddrFromRequest(r, h.behindProxy),
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

		if h.alog != nil {
			h.alog(ctx, ev)
		} else {
			var contextVals []interface{}
			CtxLogMap(ctx).Transact(func(m map[interface{}]interface{}) {
				contextVals = make([]interface{}, 0, 2*(len(m)+7))
				for k, v := range m {
					contextVals = append(contextVals, k, v)
				}
			})
			contextVals = append(contextVals,
				"App", h.appName,
				"Method", ev.Request.Method,
				"URL", ev.URL.String(),
				"UserAgent", ev.Request.UserAgent(),
				"RequestID", RequestID(ctx),
				"RemoteAddr", ev.RemoteAddr,
				"StatusCode", ev.StatusCode,
			)
			log := golog.Context(contextVals...)
			if ev.Panic != nil {
				log.Criticalf("http: panic: %v\n%s", ev.Panic, ev.StackTrace)
			} else {
				log.Infof(h.appName + " httprequest")
			}
		}

		requestEventPool.Put(ev)
		loggingResponseWriterPool.Put(logrw)
	}()

	h.h.ServeHTTP(logrw, r.WithContext(ctx))
}

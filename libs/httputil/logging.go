package httputil

import (
	"net/http"
	"runtime"
	"strings"

	"github.com/sprucehealth/backend/libs/golog"
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

type loggingHandler struct {
	h   http.Handler
	log golog.Logger
}

func LoggingHandler(h http.Handler, log golog.Logger) http.Handler {
	return &loggingHandler{
		h:   h,
		log: log,
	}
}

func (h *loggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logrw := &loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// http://en.wikipedia.org/wiki/HTTP_Strict_Transport_Security
	logrw.Header().Set("Strict-Transport-Security", "max-age=31536000")

	// Save the URL here incase it gets mangled by the time
	// the defer gets called. This can happen when suing http.StripPrefix
	// such as for static file serving.
	url := r.URL.String()
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			h.log.Context(
				"StatusCode", http.StatusInternalServerError,
				"Method", r.Method,
				"URL", url,
				"UserAgent", r.UserAgent(),
			).Criticalf("http: panic: %v\n%s", err, buf)

			if !logrw.wroteHeader {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			remoteAddr := r.RemoteAddr
			if idx := strings.LastIndex(remoteAddr, ":"); idx > 0 {
				remoteAddr = remoteAddr[:idx]
			}

			h.log.Context(
				"StatusCode", logrw.statusCode,
				"Method", r.Method,
				"URL", url,
				"RemoteAddr", remoteAddr,
				"UserAgent", r.UserAgent(),
			).LogDepthf(-1, golog.INFO, "webrequest")
		}
	}()

	h.h.ServeHTTP(logrw, r)
}

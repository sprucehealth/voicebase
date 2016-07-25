package logging

import (
	"net/http"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type ridLoggingHandler struct {
	h http.Handler
}

func (h *ridLoggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rid := httputil.RequestID(r.Context())
	logger := golog.ContextLogger(r.Context()).Context("RequestID", rid)
	h.h.ServeHTTP(w, r.WithContext(golog.WithLogger(r.Context(), logger)))
}

// NewRequestID returns a handler that adds a logger into the context that tracks the request ID
func NewRequestID(h http.Handler) http.Handler {
	return &ridLoggingHandler{
		h: h,
	}
}

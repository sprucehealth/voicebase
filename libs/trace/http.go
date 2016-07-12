package trace

import (
	"net/http"
	"strconv"

	"context"
)

const (
	httpHeaderTraceID = "X-Trace-ID"
	httpHeaderSpanID  = "X-Span-ID"
)

type HTTPHandler interface {
	ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request)
}

type HTTPHandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request)

func (h HTTPHandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h(ctx, w, r)
}

type httpContextWrapper struct {
	h   HTTPHandler
	fam string
	ids bool
}

func HTTPContext(h HTTPHandler, useRequestIDs bool, family string) http.Handler {
	return &httpContextWrapper{
		h:   h,
		fam: family,
		ids: useRequestIDs,
	}
}

func (h *httpContextWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var traceID, spanID uint64
	if h.ids {
		traceID, _ = strconv.ParseUint(r.Header.Get(httpHeaderTraceID), 10, 64)
		spanID, _ = strconv.ParseUint(r.Header.Get(httpHeaderSpanID), 10, 64)
	}
	tr := New(h.fam, r.URL.Path, traceID, spanID)
	defer tr.Finish()
	ctx := TraceContext(context.Background(), tr)
	h.h.ServeHTTP(ctx, w, r)
}

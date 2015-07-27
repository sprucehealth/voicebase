package httputil

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
)

// ContextHandler is a version of http.Handler that also takes a net/context.Context
type ContextHandler interface {
	ServeHTTP(context.Context, http.ResponseWriter, *http.Request)
}

// ContextHandlerFunc is an adapter to allow the use of ordinary functions as
// context aware HTTP handlers. If f is a function with the appropriate signature,
// ContextHandlerFunc(f) is a ContextHandler object that calls f.
type ContextHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (fn ContextHandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fn(ctx, w, r)
}

// FromContextHandler adapts a context aware handler to a normal
// http.Handler. The background context is always used.
func FromContextHandler(h ContextHandler) http.Handler {
	return fromContextHandler{h}
}

// ToContextHandler adapts a normal handler to a context are handler.
func ToContextHandler(h http.Handler) ContextHandler {
	return toContextHandler{h}
}

type fromContextHandler struct {
	h ContextHandler
}

func (a fromContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.h.ServeHTTP(context.Background(), w, r)
}

type toContextHandler struct {
	h http.Handler
}

func (a toContextHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	a.h.ServeHTTP(w, r)
}

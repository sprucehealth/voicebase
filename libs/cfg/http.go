package cfg

import (
	"context"
	"net/http"
)

// contextKeyType creates a unique type to be used in the request context
type contextKeyType int

var contextKey contextKeyType

// HTTPHandler returns a wrapped handler that sets a snapshot of
// the cfg store in the request context.
func HTTPHandler(h http.Handler, store Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey, store.Snapshot())))
	})
}

// Context returns the Snapshot of config values for an HTTP request.
func Context(ctx context.Context) Snapshot {
	return ctx.Value(contextKey).(Snapshot)
}

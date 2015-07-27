package cfg

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/libs/httputil"
)

// contextKeyType creates a unique type to be used in the request context
type contextKeyType int

var contextKey contextKeyType

// HTTPHandler returns a wrapped handler that sets a snapshot of
// the cfg store in the request context.
func HTTPHandler(h httputil.ContextHandler, store Store) httputil.ContextHandler {
	return httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(context.WithValue(ctx, contextKey, store.Snapshot()), w, r)
	})
}

// Context returns the Snapshot of config values for an HTTP request.
func Context(ctx context.Context) Snapshot {
	return ctx.Value(contextKey).(Snapshot)
}

package cfg

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
)

// contextKeyType creates a unique type to be used in the request context
type contextKeyType int

var contextKey contextKeyType

// HTTPHandler returns a wrapped handler that sets a snapshot of
// the cfg store in the request context.
func HTTPHandler(h http.Handler, store Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		context.Set(r, contextKey, store.Snapshot())
		h.ServeHTTP(w, r)
	})
}

func Context(r *http.Request) Snapshot {
	return context.Get(r, contextKey).(Snapshot)
}

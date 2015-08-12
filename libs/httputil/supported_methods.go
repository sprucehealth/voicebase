package httputil

import (
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
)

type supportedMethods struct {
	methods []string
	handler http.Handler
}

type contextSupportedMethods struct {
	methods []string
	handler ContextHandler
}

// SupportedMethods wraps an HTTP handler, and before a request is
// passed to the handler the method is checked against the list provided.
// If it does not match one of the expected methods then StatusMethodNotAllowed
// status is returned along with a list of allowed methods in the "Allow"
// HTTP header.
func SupportedMethods(h http.Handler, methods ...string) http.Handler {
	return &supportedMethods{
		methods: methods,
		handler: h,
	}
}

// ContextSupportedMethods wraps an HTTP handler, and before a request is
// passed to the handler the method is checked against the list provided.
// If it does not match one of the expected methods then StatusMethodNotAllowed
// status is returned along with a list of allowed methods in the "Allow"
// HTTP header.
//
// TODO: this will be merged with SupportedMethod once we're using context
// aware handlers everywhere.
func ContextSupportedMethods(h ContextHandler, methods ...string) ContextHandler {
	return &contextSupportedMethods{
		methods: methods,
		handler: h,
	}
}

func (sm *supportedMethods) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, m := range sm.methods {
		if r.Method == m {
			sm.handler.ServeHTTP(w, r)
			return
		}
	}
	SupportedMethodsResponse(w, r, sm.methods)
}

func (sm *contextSupportedMethods) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	for _, m := range sm.methods {
		if r.Method == m {
			sm.handler.ServeHTTP(ctx, w, r)
			return
		}
	}
	SupportedMethodsResponse(w, r, sm.methods)
}

// SupportedMethodsResponse writes a "method not allowed" response with the provided list
// of methods in the "Allow" header. For "OPTIONS" requests the response status is 200 OK
// otherwise it's 405 Method Not Allowed.
func SupportedMethodsResponse(w http.ResponseWriter, r *http.Request, methods []string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	if r.Method == Options {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

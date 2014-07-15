package httputil

import (
	"net/http"
	"sort"
	"strings"
)

type supportedMethods struct {
	methods []string
	handler http.Handler
}

// SupportedMethods wraps an HTTP handler, and before a request is
// passed to the handler the method is checked against the list provided.
// If it does not match one of the expected methods then StatusMethodNotAllowed
// status is returned along with a list of allowed methods in the "Allow"
// HTTP header.
func SupportedMethods(h http.Handler, methods []string) http.Handler {
	sort.Strings(methods)
	return &supportedMethods{
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

	w.Header().Set("Allow", strings.Join(sm.methods, ", "))
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

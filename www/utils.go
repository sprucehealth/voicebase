package www

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type supportedMethods struct {
	methods []string
	handler http.Handler
}

func SupportedMethodsFilter(h http.Handler, methods []string) http.Handler {
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

// validateRedirectURL makes sure that a user provided URL that will be
// used for a redirect (such as 'next' during login) is valid and safe.
// It also optionally rewrites the URL when appropriate.
func validateRedirectURL(urlString string) (string, bool) {
	u, err := url.Parse(urlString)
	if err != nil {
		println(err.Error())
		return "", false
	}
	path := u.Path
	if len(path) == 0 || path[0] != '/' {
		return "", false
	}
	// TODO: what else needs to be checked?
	return path, true
}

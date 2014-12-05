package router

import (
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/environment"
)

// muxWithRegisteredPaths tracks the registerd paths
// in the test environment.
type muxWithRegisteredPaths struct {
	http.ServeMux
	registeredPatterns []string
}

func newMux() *muxWithRegisteredPaths {
	m := &muxWithRegisteredPaths{
		ServeMux: *http.NewServeMux(),
	}

	// add a handler for querying the comprehensive list of paths
	// that the restapi server supports
	// Note that this handler should only process in the test environment
	if environment.IsTest() {
		m.ServeMux.Handle("/listpaths", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiservice.WriteJSON(w, map[string]interface{}{
				"paths": m.registeredPatterns,
			})
		}))
	}

	return m
}

func (m *muxWithRegisteredPaths) Handle(pattern string, handler http.Handler) {
	if environment.IsTest() {
		m.registeredPatterns = append(m.registeredPatterns, pattern)
	}
	m.ServeMux.Handle(pattern, handler)

}

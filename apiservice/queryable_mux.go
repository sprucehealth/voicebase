package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/environment"
)

type QueryableMux interface {
	http.Handler
	IsSupportedPath(string) bool
	SupportedPaths() []string
	Handle(string, http.Handler)
}

// QueryableMux tracks the registerd paths
// in the test environment.
type queryableMux struct {
	http.ServeMux
	registeredPatterns map[string]bool
}

func NewQueryableMux() *queryableMux {
	m := &queryableMux{
		ServeMux:           *http.NewServeMux(),
		registeredPatterns: make(map[string]bool),
	}

	// add a handler for querying the comprehensive list of paths
	// that the restapi server supports
	// Note that this handler should only process in the test environment
	if environment.IsTest() {
		m.ServeMux.Handle("/listpaths", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			registeredPatternsList := make([]string, 0)
			for k, _ := range m.registeredPatterns {
				registeredPatternsList = append(registeredPatternsList, k)
			}
			WriteJSON(w, map[string]interface{}{
				"paths": registeredPatternsList,
			})
		}))
	}

	return m
}

func (m *queryableMux) Handle(pattern string, handler http.Handler) {
	m.registeredPatterns[pattern] = true
	m.ServeMux.Handle(pattern, handler)
}

func (m *queryableMux) IsSupportedPath(path string) bool {
	_, ok := m.registeredPatterns[path]
	return ok
}

func (m *queryableMux) SupportedPaths() []string {
	paths := make([]string, 0)
	for k, _ := range m.registeredPatterns {
		paths = append(paths, k)
	}
	return paths
}

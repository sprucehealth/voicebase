package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/libs/httputil"
)

type supportedMethodsHandler struct {
	originalHandler         http.Handler
	supportedMethodsHandler http.Handler
}

// SupportedMethods is a wrapper for the supporedMethods in the httputil package
// to support the NonAuthenticated and Authorized interfaces.
// FIX: This is needed because the conformity to the interfaces is lost when the handler is
// wrapped in the httputil SupportedMethods handler. This current solution is not scalable in that
// if we find another useful http wrapper in httputil, then we are going to have to make sure that that
// wrapper also correctly conforms to the two interfaces. Need to figure out a better way.
func SupportedMethods(h http.Handler, methods []string) http.Handler {
	return &supportedMethodsHandler{
		originalHandler:         h,
		supportedMethodsHandler: httputil.SupportedMethods(h, methods),
	}
}

func (s *supportedMethodsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.supportedMethodsHandler.ServeHTTP(w, r)
}

func (s *supportedMethodsHandler) IsAuthorized(r *http.Request) (bool, error) {
	return s.originalHandler.(Authorized).IsAuthorized(r)
}

func (s *supportedMethodsHandler) NonAuthenticated() bool {
	n, ok := s.originalHandler.(NonAuthenticated)
	return ok && n.NonAuthenticated()
}

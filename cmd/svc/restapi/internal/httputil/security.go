package httputil

import (
	"net/http"
)

type securityHandler struct {
	h http.Handler
}

// SecurityHandler wraps a handler and sets security headers on the response.
// http://en.wikipedia.org/wiki/HTTP_Strict_Transport_Security
func SecurityHandler(h http.Handler) http.Handler {
	return securityHandler{h}
}

func (h securityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	h.h.ServeHTTP(w, r)
}

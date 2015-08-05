package httputil

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
)

type securityHandler struct {
	h ContextHandler
}

// SecurityHandler wraps a handler and sets security headers on the response.
// http://en.wikipedia.org/wiki/HTTP_Strict_Transport_Security
func SecurityHandler(h ContextHandler) ContextHandler {
	return securityHandler{h}
}

func (h securityHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	h.h.ServeHTTP(ctx, w, r)
}

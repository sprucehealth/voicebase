package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/libs/mux"
)

// InitRoutes registers the media service handlers on the provided mux
func InitRoutes(
	r *mux.Router,
	dal dal.DAL,
) {
	r.Handle(`/{token:\d+}`, &orgCodeHandler{dal: dal})
	r.Handle(`/robots.txt`, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("User-agent: *\nDisallow: /\n"))
	}))
	// TODO: re-enable once iOS issue for universal links handling fixed.
	// r.Handle(`/apple-app-site-association`, &appleDeeplinkHandler{})
}

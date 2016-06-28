package handlers

import (
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/libs/mux"
)

// InitRoutes registers the media service handlers on the provided mux
func InitRoutes(
	r *mux.Router,
	dal dal.DAL,
) {
	r.Handle(`/{token:\d+}`, &orgCodeHandler{dal: dal})
}

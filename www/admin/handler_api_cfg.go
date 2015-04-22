package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/httputil"
)

type cfgHandler struct {
	cfg cfg.Store
}

func NewCFGHandler(cfg cfg.Store) http.Handler {
	return httputil.SupportedMethods(&cfgHandler{
		cfg: cfg,
	}, []string{httputil.Get, httputil.Patch})
}

func (h *cfgHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

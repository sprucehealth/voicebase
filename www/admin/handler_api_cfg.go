package admin

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type cfgHandler struct {
	cfg cfg.Store
}

type cfgResponse struct {
	Snapshot cfg.Snapshot             `json:"snapshot"`
	Defs     map[string]*cfg.ValueDef `json:"defs"`
}

type cfgUpdate struct {
	Snapshot cfg.Snapshot `json:"values"`
}

func NewCFGHandler(cfg cfg.Store) http.Handler {
	return httputil.SupportedMethods(&cfgHandler{
		cfg: cfg,
	}, []string{httputil.Get, httputil.Patch})
}

func (h *cfgHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		httputil.JSONResponse(w, http.StatusOK, &cfgResponse{
			Snapshot: h.cfg.Snapshot(),
			Defs:     h.cfg.Defs(),
		})
	case httputil.Patch:
		var update cfgUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			www.APIBadRequestError(w, r, "Failed to decode body")
			return
		}
		values := update.Snapshot.Values()
		if err := cfg.CoerceValues(h.cfg.Defs(), values); err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		if err := h.cfg.Update(values); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		httputil.JSONResponse(w, http.StatusOK, &cfgResponse{
			Snapshot: h.cfg.Snapshot(),
			Defs:     h.cfg.Defs(),
		})
	}
}

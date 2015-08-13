package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type layoutVersionHandler struct {
	dataAPI api.DataAPI
}

type layoutVersionResponse struct {
	Items []*api.LayoutVersionInfo `json:"items"`
}

func newLayoutVersionHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&layoutVersionHandler{dataAPI: dataAPI}, httputil.Get)
}

func (h *layoutVersionHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	layoutVersions, err := h.dataAPI.LayoutVersions()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &layoutVersionResponse{Items: layoutVersions})
}

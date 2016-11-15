package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
)

type layoutVersionHandler struct {
	dataAPI api.DataAPI
}

type layoutVersionResponse struct {
	Items []*api.LayoutVersionInfo `json:"items"`
}

func newLayoutVersionHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&layoutVersionHandler{dataAPI: dataAPI}, httputil.Get)
}

func (h *layoutVersionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	layoutVersions, err := h.dataAPI.LayoutVersions()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &layoutVersionResponse{Items: layoutVersions})
}

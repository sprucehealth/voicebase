package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type layoutVersionHandler struct {
	dataAPI api.DataAPI
}

func NewLayoutVersionHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&layoutVersionHandler{dataAPI: dataAPI}, []string{"GET"})
}

func (h *layoutVersionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	versionMapping, err := h.dataAPI.LayoutVersionMapping()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	www.JSONResponse(w, r, http.StatusOK, versionMapping)
}

package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type layoutVersionHandler struct {
	dataAPI api.DataAPI
}

type layoutVersionGETResponse map[string]map[string][]string

func NewLayoutVersionHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			&layoutVersionHandler{
				dataAPI: dataAPI,
			}, []string{api.ADMIN_ROLE}), []string{"GET"})
}

func (h *layoutVersionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get a map of layout versions and info
	versionMapping, err := h.dataAPI.LayoutVersionMapping()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	www.JSONResponse(w, r, http.StatusOK, versionMapping)
}

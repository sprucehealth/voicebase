package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
)

type resourceGuidesAPIHandler struct {
	dataAPI api.DataAPI
}

func newResourceGuidesAPIHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&resourceGuidesAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Patch)
}

func (h *resourceGuidesAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := www.MustCtxAccount(ctx)
	if r.Method == "PATCH" {
		audit.LogAction(account.ID, "AdminAPI", "UpdateResourceGuide", map[string]interface{}{"guide_id": id})

		var update api.ResourceGuideUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		if err := h.dataAPI.UpdateResourceGuide(id, &update); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		httputil.JSONResponse(w, http.StatusOK, true)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "GetResourceGuide", map[string]interface{}{"guide_id": id})

	guide, err := h.dataAPI.GetResourceGuide(id)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, guide)
}

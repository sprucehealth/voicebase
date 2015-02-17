package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type resourceGuidesAPIHandler struct {
	dataAPI api.DataAPI
}

func NewResourceGuidesAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&resourceGuidesAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *resourceGuidesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	if r.Method == "POST" {
		audit.LogAction(account.ID, "AdminAPI", "UpdateResourceGuide", map[string]interface{}{"guide_id": id})

		var guide common.ResourceGuide
		if err := json.NewDecoder(r.Body).Decode(&guide); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		guide.ID = id
		if err := h.dataAPI.UpdateResourceGuide(&guide); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		httputil.JSONResponse(w, http.StatusOK, true)
		return
	} else {
		audit.LogAction(account.ID, "AdminAPI", "GetResourceGuide", map[string]interface{}{"guide_id": id})
	}

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

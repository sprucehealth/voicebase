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

type emailTemplatesHandler struct {
	dataAPI api.DataAPI
}

func NewEmailTemplatesHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&emailTemplatesHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "PUT"})
}

func (h *emailTemplatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)

	if r.Method == "PUT" {
		audit.LogAction(account.ID, "AdminAPI", "UpdateEmailTemplate", map[string]interface{}{"template_id": id})

		var update api.EmailTemplateUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		if err := h.dataAPI.UpdateEmailTemplate(id, &update); err == api.NoRowsError {
			www.APINotFound(w, r)
			return
		} else if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, true)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "GetEmailTemplate", map[string]interface{}{"template_id": id})

	tmpl, err := h.dataAPI.GetEmailTemplate(id)
	if err == api.NoRowsError {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	www.JSONResponse(w, r, http.StatusOK, tmpl)
}

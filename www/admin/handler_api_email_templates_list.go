package admin

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/www"
)

type emailTemplatesListHandler struct {
	dataAPI api.DataAPI
}

func NewEmailTemplatesListHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&emailTemplatesListHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *emailTemplatesListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

	if r.Method == "POST" {
		audit.LogAction(account.ID, "AdminAPI", "CreateEmailTemplate", nil)

		var req common.EmailTemplate
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		id, err := h.dataAPI.CreateEmailTemplate(&req)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, id)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "ListEmailTemplates", nil)

	templates, err := h.dataAPI.ListEmailTemplates(r.FormValue("type"))
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	www.JSONResponse(w, r, http.StatusOK, templates)
}

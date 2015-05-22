package admin

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type schedMessageTemplatesListAPIHandler struct {
	dataAPI api.DataAPI
}

func NewSchedMessageTemplatesListAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&schedMessageTemplatesListAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Post)
}

func (h *schedMessageTemplatesListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

	if r.Method == "POST" {
		audit.LogAction(account.ID, "AdminAPI", "CreateScheduledMessageTemplate", nil)

		var rep common.ScheduledMessageTemplate
		if err := json.NewDecoder(r.Body).Decode(&rep); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		if err := h.dataAPI.CreateScheduledMessageTemplate(&rep); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		httputil.JSONResponse(w, http.StatusOK, rep.ID)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "ListScheduledMessageTemplates", nil)

	templates, err := h.dataAPI.ListScheduledMessageTemplates()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, templates)
}

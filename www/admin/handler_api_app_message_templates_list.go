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

type appMessageTemplatesListAPIHandler struct {
	dataAPI api.DataAPI
}

func NewAppMessageTemplatesListAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&appMessageTemplatesAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *appMessageTemplatesListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

	if r.Method == "POST" {
		audit.LogAction(account.ID, "AdminAPI", "CreateScheduledMessageTemplate", nil)

		var rep common.ScheduledMessageTemplate
		if err := json.NewDecoder(r.Body).Decode(&rep); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		rep.CreatorAccountID = account.ID

		if err := h.dataAPI.CreateScheduledMessageTemplate(&rep); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, rep.ID)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "ListScheduledMessageTemplates", nil)

	templates, err := h.dataAPI.ListScheduledMessageTemplates()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	www.JSONResponse(w, r, http.StatusOK, templates)
}

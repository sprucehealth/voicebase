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

type schedMessageTemplatesAPIHandler struct {
	dataAPI api.DataAPI
}

func NewSchedMessageTemplatesAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&schedMessageTemplatesAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "PUT", "DELETE"})
}

func (h *schedMessageTemplatesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)

	if r.Method == "PUT" {
		audit.LogAction(account.ID, "AdminAPI", "UpdateScheduledMessageTemplate", map[string]interface{}{"template_id": id})

		var updateReq common.ScheduledMessageTemplate
		if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		schedMessageTemplate := &common.ScheduledMessageTemplate{
			ID:             id,
			Name:           updateReq.Name,
			Message:        updateReq.Message,
			Event:          updateReq.Event,
			SchedulePeriod: updateReq.SchedulePeriod,
		}
		if err := h.dataAPI.UpdateScheduledMessageTemplate(schedMessageTemplate); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, true)
		return
	} else if r.Method == "DELETE" {
		audit.LogAction(account.ID, "AdminAPI", "DeleteScheduledMessageTemplate", map[string]interface{}{"template_id": id})

		if err := h.dataAPI.DeleteScheduledMessageTemplate(id); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, true)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "GetScheduledMessageTemplate", map[string]interface{}{"template_id": id})

	template, err := h.dataAPI.ScheduledMessageTemplate(id)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	www.JSONResponse(w, r, http.StatusOK, template)
}

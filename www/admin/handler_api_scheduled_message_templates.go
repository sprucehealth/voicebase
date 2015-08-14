package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type schedMessageTemplatesAPIHandler struct {
	dataAPI api.DataAPI
}

func newSchedMessageTemplatesAPIHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&schedMessageTemplatesAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Put, httputil.Delete)
}

func (h *schedMessageTemplatesAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(ctx)

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

		httputil.JSONResponse(w, http.StatusOK, true)
		return
	} else if r.Method == "DELETE" {
		audit.LogAction(account.ID, "AdminAPI", "DeleteScheduledMessageTemplate", map[string]interface{}{"template_id": id})

		if err := h.dataAPI.DeleteScheduledMessageTemplate(id); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		httputil.JSONResponse(w, http.StatusOK, true)
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

	httputil.JSONResponse(w, http.StatusOK, template)
}

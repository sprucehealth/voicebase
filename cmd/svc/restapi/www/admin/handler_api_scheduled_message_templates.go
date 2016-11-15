package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/mux"
)

type schedMessageTemplatesAPIHandler struct {
	dataAPI api.DataAPI
}

func newSchedMessageTemplatesAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&schedMessageTemplatesAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Put, httputil.Delete)
}

func (h *schedMessageTemplatesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r.Context())["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(r.Context())

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

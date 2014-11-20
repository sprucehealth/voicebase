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

type emailSendersListHandler struct {
	dataAPI api.DataAPI
}

func NewEmailSendersListHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&emailSendersListHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *emailSendersListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

	if r.Method == "POST" {
		audit.LogAction(account.ID, "AdminAPI", "CreateEmailSender", nil)

		var req common.EmailSender
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		id, err := h.dataAPI.CreateEmailSender(req.Name, req.Email)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, id)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "ListEmailSenders", nil)

	senders, err := h.dataAPI.ListEmailSenders()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	www.JSONResponse(w, r, http.StatusOK, senders)
}

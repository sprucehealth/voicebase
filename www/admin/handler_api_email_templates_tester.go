package admin

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"reflect"
	"strconv"

	"github.com/sprucehealth/backend/email"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type emailTemplatesTestHandler struct {
	emailService email.Service
	dataAPI      api.DataAPI
}

type emailTemplatesTestRequest struct {
	To      string          `json:"to"`
	Context json.RawMessage `json:"context"`
}

func NewEmailTemplatesTestHandler(emailService email.Service, dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&emailTemplatesTestHandler{
		emailService: emailService,
		dataAPI:      dataAPI,
	}, []string{"POST"})
}

func (h *emailTemplatesTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	var req emailTemplatesTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}
	if req.To == "" {
		www.APIBadRequestError(w, r, "'to' is required")
		return
	}
	to, err := mail.ParseAddress(req.To)
	if err != nil {
		www.APIBadRequestError(w, r, "'to' is invalid: "+err.Error())
		return
	}
	if req.Context == nil {
		www.APIBadRequestError(w, r, "'context' is required")
		return
	}

	tmpl, err := h.dataAPI.GetEmailTemplate(id)
	if err == api.NoRowsError {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "TestEmailTemplate",
		map[string]interface{}{
			"template_id": id,
			"to_email":    req.To,
		})

	emailType := email.Types[tmpl.Type]
	if emailType == nil {
		golog.Errorf("Email type %s unknown when trying to send test", tmpl.Type)
		www.APINotFound(w, r)
		return
	}

	ctx := reflect.New(reflect.TypeOf(emailType.TestContext)).Interface()
	if err := json.Unmarshal(req.Context, &ctx); err != nil {
		www.APIBadRequestError(w, r, "Invalid context: "+err.Error())
		return
	}

	if err := h.emailService.SendTemplate(to, id, ctx); err != nil {
		www.APIBadRequestError(w, r, "Failed to send: "+err.Error())
		return
	}

	www.JSONResponse(w, r, http.StatusOK, true)
}

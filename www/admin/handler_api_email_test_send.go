package admin

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/email/campaigns"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mandrill"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/www"
)

type emailTestSendHandler struct {
	emailService email.Service
	signer       *sig.Signer
	webDomain    string
}

type emailTestSendRequest struct {
	Type string `json:"type"`
}

type emailTestSendResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func NewEmailTestSendHandler(emailService email.Service, signer *sig.Signer, webDomain string) http.Handler {
	return httputil.SupportedMethods(&emailTestSendHandler{
		emailService: emailService,
		signer:       signer,
		webDomain:    webDomain,
	}, httputil.Post)
}

func (h *emailTestSendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req emailTestSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, "Failed to parse request body")
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "SendTestEmail", map[string]interface{}{"type": req.Type})

	vars := campaigns.VarsForAccount(account.ID, req.Type, h.signer, h.webDomain)
	if _, err := h.emailService.Send([]int64{account.ID}, req.Type, map[int64][]mandrill.Var{account.ID: vars}, &mandrill.Message{}, 0); err != nil {
		httputil.JSONResponse(w, http.StatusOK, &emailTestSendResponse{Success: false, Error: err.Error()})
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &emailTestSendResponse{Success: true})
}

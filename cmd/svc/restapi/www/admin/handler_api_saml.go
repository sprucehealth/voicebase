package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/saml"
)

type samlAPIHandler struct {
}

type samlRequest struct {
	SAML string `json:"saml"`
}

func newSAMLAPIHandler() http.Handler {
	return httputil.SupportedMethods(&samlAPIHandler{}, httputil.Post)
}

func (h *samlAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(r.Context())
	audit.LogAction(account.ID, "AdminAPI", "SAMLTransform", nil)

	var req samlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, "Failed to decode JSON body")
		return
	}

	intake, err := saml.Parse(strings.NewReader(req.SAML))
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	httputil.JSONResponse(w, http.StatusOK, &struct {
		Intake *saml.Intake `json:"intake,omitempty"`
		Error  string       `json:"error,omitempty"`
	}{
		Intake: intake,
		Error:  errStr,
	})
}

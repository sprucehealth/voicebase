package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/www"
)

type samlAPIHandler struct {
}

type samlRequest struct {
	SAML string `json:"saml"`
}

func newSAMLAPIHandler() httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&samlAPIHandler{}, httputil.Post)
}

func (h *samlAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
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

package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type emailTypesListHandler struct {
}

func NewEmailTypesListHandler() http.Handler {
	return httputil.SupportedMethods(&emailTypesListHandler{}, []string{"GET"})
}

func (h *emailTypesListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "ListEmailTypes", nil)
	httputil.JSONResponse(w, http.StatusOK, email.Types)
}

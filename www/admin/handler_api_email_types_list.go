package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
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
	www.JSONResponse(w, r, http.StatusOK, email.Types)
}

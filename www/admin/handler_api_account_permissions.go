package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type accountAvailablePermissionsAPIHandler struct {
	authAPI api.AuthAPI
}

func NewAccountAvailablePermissionsAPIHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(&accountAvailablePermissionsAPIHandler{
		authAPI: authAPI,
	}, []string{"GET"})
}

func (h *accountAvailablePermissionsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "ListAvailableAccountPermissions", nil)

	perms, err := h.authAPI.AvailableAccountPermissions()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Permissions []string `json:"permissions"`
	}{
		Permissions: perms,
	})
}

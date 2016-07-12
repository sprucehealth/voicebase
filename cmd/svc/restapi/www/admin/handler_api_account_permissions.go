package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
)

type accountAvailablePermissionsAPIHandler struct {
	authAPI api.AuthAPI
}

func newAccountAvailablePermissionsAPIHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(&accountAvailablePermissionsAPIHandler{
		authAPI: authAPI,
	}, httputil.Get)
}

func (h *accountAvailablePermissionsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(r.Context())
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

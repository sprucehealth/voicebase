package admin

import (
	"net/http"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
)

type accountAvailablePermissionsAPIHandler struct {
	authAPI api.AuthAPI
}

func newAccountAvailablePermissionsAPIHandler(authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&accountAvailablePermissionsAPIHandler{
		authAPI: authAPI,
	}, httputil.Get)
}

func (h *accountAvailablePermissionsAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
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

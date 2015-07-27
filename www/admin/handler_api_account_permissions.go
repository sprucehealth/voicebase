package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type accountAvailablePermissionsAPIHandler struct {
	authAPI api.AuthAPI
}

func newAccountAvailablePermissionsAPIHandler(authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&accountAvailablePermissionsAPIHandler{
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

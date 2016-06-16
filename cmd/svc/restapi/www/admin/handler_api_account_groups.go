package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type accountAvailableGroupsAPIHandler struct {
	authAPI api.AuthAPI
}

func newAccountAvailableGroupsAPIHandler(authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&accountAvailableGroupsAPIHandler{
		authAPI: authAPI,
	}, httputil.Get)
}

func (h *accountAvailableGroupsAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "ListAvailableAccountGroups", nil)

	groups, err := h.authAPI.AvailableAccountGroups(r.FormValue("with_perms") != "")
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Groups []*common.AccountGroup `json:"groups"`
	}{
		Groups: groups,
	})
}

package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/www"
)

type accountAvailableGroupsAPIHandler struct {
	authAPI api.AuthAPI
}

func NewAccountAvailableGroupsAPIHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(&accountAvailableGroupsAPIHandler{
		authAPI: authAPI,
	}, []string{"GET"})
}

func (h *accountAvailableGroupsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "ListAvailableAccountGroups", nil)

	groups, err := h.authAPI.AvailableAccountGroups(r.FormValue("with_perms") != "")
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	www.JSONResponse(w, r, http.StatusOK, &struct {
		Groups []*common.AccountGroup `json:"groups"`
	}{
		Groups: groups,
	})
}

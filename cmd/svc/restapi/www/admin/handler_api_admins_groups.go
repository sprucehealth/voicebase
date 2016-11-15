package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/mux"
)

type adminsGroupsAPIHandler struct {
	authAPI api.AuthAPI
}

func newAdminsGroupsAPIHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(&adminsGroupsAPIHandler{
		authAPI: authAPI,
	}, httputil.Get, httputil.Post)
}

func (h *adminsGroupsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseInt(mux.Vars(r.Context())["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(r.Context())

	if r.Method == httputil.Post {
		// Use a string key because JSON
		var groups map[string]bool
		if err := json.NewDecoder(r.Body).Decode(&groups); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		groupsUpdate := make(map[int64]bool, len(groups))
		for idStr, state := range groups {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				www.APIInternalError(w, r, err)
				return
			}
			groupsUpdate[id] = state
		}

		audit.LogAction(account.ID, "AdminAPI", "UpdateAdminGroups", map[string]interface{}{"param_account_id": accountID, "groups": groups})

		if err := h.authAPI.UpdateGroupsForAccount(accountID, groupsUpdate); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		httputil.JSONResponse(w, http.StatusOK, true)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "GetAdminGroups", map[string]interface{}{"param_account_id": accountID})

	// Verify account exists and is the correct role
	acc, err := h.authAPI.GetAccount(accountID)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	} else if acc.Role != api.RoleAdmin {
		www.APINotFound(w, r)
		return
	}

	groups, err := h.authAPI.GroupsForAccount(accountID)
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

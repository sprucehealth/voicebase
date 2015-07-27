package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
)

type accountHandler struct {
	authAPI api.AuthAPI
}

type accountUpdateRequest struct {
	Email            *string `json:"email"`
	TwoFactorEnabled *bool   `json:"two_factor_enabled"`
}

type accountResponse struct {
	Account *common.Account `json:"account"`
}

func newAccountHandler(authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&accountHandler{
		authAPI: authAPI,
	}, httputil.Get, httputil.Patch)
}

func (h *accountHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	reqAccountID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}
	reqAccount, err := h.authAPI.GetAccount(reqAccountID)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := www.MustCtxAccount(ctx)
	perms := www.MustCtxPermissions(ctx)

	if r.Method == httputil.Patch {
		if !accountWriteAccess(reqAccount, perms) {
			audit.LogAction(account.ID, "AdminAPI", "UpdateAccount", map[string]interface{}{"denied": true, "req_account_id": reqAccountID})
			www.APIForbidden(w, r)
			return
		}
		audit.LogAction(account.ID, "AdminAPI", "UpdateAccount", map[string]interface{}{"req_account_id": reqAccountID})

		var req accountUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		if err := h.authAPI.UpdateAccount(reqAccountID, req.Email, req.TwoFactorEnabled); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		if req.Email != nil {
			reqAccount.Email = *req.Email
		}
		if req.TwoFactorEnabled != nil {
			reqAccount.TwoFactorEnabled = *req.TwoFactorEnabled
		}
		httputil.JSONResponse(w, http.StatusOK, &accountResponse{Account: reqAccount})
		return
	}

	if !accountReadAccess(reqAccount, perms) {
		audit.LogAction(account.ID, "AdminAPI", "GetAccount", map[string]interface{}{"denied": true, "req_account_id": reqAccountID})
		www.APIForbidden(w, r)
		return
	}
	audit.LogAction(account.ID, "AdminAPI", "GetAccount", map[string]interface{}{"req_account_id": reqAccountID})

	httputil.JSONResponse(w, http.StatusOK, &accountResponse{Account: reqAccount})
}

func accountReadAccess(account *common.Account, perms www.Permissions) bool {
	if perms.Has(PermDoctorsView) &&
		(account.Role == api.RoleDoctor || account.Role == api.RoleCC) {
		return true
	}
	if perms.Has(PermAdminAccountsView) && account.Role == api.RoleAdmin {
		return true
	}
	return false
}

func accountWriteAccess(account *common.Account, perms www.Permissions) bool {
	if perms.Has(PermDoctorsEdit) &&
		(account.Role == api.RoleDoctor || account.Role == api.RoleCC) {
		return true
	}
	if perms.Has(PermAdminAccountsEdit) && account.Role == api.RoleAdmin {
		return true
	}
	return false
}

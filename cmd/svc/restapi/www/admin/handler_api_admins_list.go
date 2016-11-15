package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
)

type adminsListAPIHandler struct {
	authAPI api.AuthAPI
}

func newAdminsListAPIHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(&adminsListAPIHandler{
		authAPI: authAPI,
	}, httputil.Get)
}

func (h *adminsListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")

	account := www.MustCtxAccount(r.Context())
	audit.LogAction(account.ID, "AdminAPI", "ListAdmins", map[string]interface{}{"query": query})

	var accounts []*common.Account

	if query != "" {
		// TODO: for now just search by exact email
		if a, err := h.authAPI.AccountForEmail(query); err == nil {
			if a.Role == api.RoleAdmin {
				accounts = append(accounts, a)
			}
		} else if !api.IsErrNotFound(err) && err != api.ErrLoginDoesNotExist {
			www.APIInternalError(w, r, err)
			return
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Accounts []*common.Account `json:"accounts"`
	}{
		Accounts: accounts,
	})
}

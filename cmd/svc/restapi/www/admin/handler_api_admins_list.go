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

type adminsListAPIHandler struct {
	authAPI api.AuthAPI
}

func newAdminsListAPIHandler(authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&adminsListAPIHandler{
		authAPI: authAPI,
	}, httputil.Get)
}

func (h *adminsListAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")

	account := www.MustCtxAccount(ctx)
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
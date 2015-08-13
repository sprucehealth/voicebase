package admin

import (
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

type adminsAPIHandler struct {
	authAPI api.AuthAPI
}

func newAdminsAPIHandler(authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&adminsAPIHandler{
		authAPI: authAPI,
	}, httputil.Get)
}

func (h *adminsAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "GetAdmin", map[string]interface{}{"param_account_id": accountID})

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

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Account *common.Account `json:"account"`
	}{
		Account: acc,
	})
}

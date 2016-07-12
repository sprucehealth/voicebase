package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
)

type adminsAPIHandler struct {
	authAPI api.AuthAPI
}

func newAdminsAPIHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(&adminsAPIHandler{
		authAPI: authAPI,
	}, httputil.Get)
}

func (h *adminsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseInt(mux.Vars(r.Context())["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(r.Context())
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

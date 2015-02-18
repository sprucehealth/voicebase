package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type adminsAPIHandler struct {
	authAPI api.AuthAPI
}

func NewAdminsAPIHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(&adminsAPIHandler{
		authAPI: authAPI,
	}, []string{"GET"})
}

func (h *adminsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "GetAdmin", map[string]interface{}{"param_account_id": accountID})

	acc, err := h.authAPI.GetAccount(accountID)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	} else if acc.Role != api.ADMIN_ROLE {
		www.APINotFound(w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Account *common.Account `json:"account"`
	}{
		Account: acc,
	})
}

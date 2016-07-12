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

type accountPhonesListHandler struct {
	authAPI api.AuthAPI
}

func newAccountPhonesListHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(&accountPhonesListHandler{
		authAPI: authAPI,
	}, httputil.Get)
}

func (h *accountPhonesListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqAccountID, err := strconv.ParseInt(mux.Vars(r.Context())["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
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

	account := www.MustCtxAccount(r.Context())
	perms := www.MustCtxPermissions(r.Context())
	if !accountReadAccess(reqAccount, perms) {
		audit.LogAction(account.ID, "AdminAPI", "ListAccountPhoneNumbers", map[string]interface{}{"denied": true, "req_account_id": reqAccountID})
		www.APIForbidden(w, r)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "ListAccountPhoneNumbers", map[string]interface{}{"req_account_id": reqAccountID})

	numbers, err := h.authAPI.GetPhoneNumbersForAccount(reqAccountID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Numbers []*common.PhoneNumber `json:"numbers"`
	}{
		Numbers: numbers,
	})
}

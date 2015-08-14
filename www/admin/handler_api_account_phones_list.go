package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type accountPhonesListHandler struct {
	authAPI api.AuthAPI
}

func newAccountPhonesListHandler(authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&accountPhonesListHandler{
		authAPI: authAPI,
	}, httputil.Get)
}

func (h *accountPhonesListHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	reqAccountID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
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

	account := www.MustCtxAccount(ctx)

	perms := www.MustCtxPermissions(ctx)
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

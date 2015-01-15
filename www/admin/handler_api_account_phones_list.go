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

type accountPhonesListHandler struct {
	authAPI api.AuthAPI
}

func NewAccountPhonesListHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(&accountPhonesListHandler{
		authAPI: authAPI,
	}, []string{"GET"})
}

func (h *accountPhonesListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqAccountID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
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

	account := context.Get(r, www.CKAccount).(*common.Account)

	perms := context.Get(r, www.CKPermissions).(www.Permissions)
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

	www.JSONResponse(w, r, http.StatusOK, &struct {
		Numbers []*common.PhoneNumber `json:"numbers"`
	}{
		Numbers: numbers,
	})
}

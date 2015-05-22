package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/schedmsg"
	"github.com/sprucehealth/backend/www"
)

type schedMessageEventsListAPIHandler struct{}

func NewSchedMessageEventsListAPIHandler() http.Handler {
	return httputil.SupportedMethods(&schedMessageEventsListAPIHandler{},
		httputil.Get)
}

func (h *schedMessageEventsListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "ListSchedMessageEvents", nil)
	httputil.JSONResponse(w, http.StatusOK, schedmsg.Events)
}

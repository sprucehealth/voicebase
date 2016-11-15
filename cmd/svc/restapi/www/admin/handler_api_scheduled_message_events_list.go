package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/schedmsg"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
)

type schedMessageEventsListAPIHandler struct{}

func newSchedMessageEventsListAPIHandler() http.Handler {
	return httputil.SupportedMethods(&schedMessageEventsListAPIHandler{},
		httputil.Get)
}

func (h *schedMessageEventsListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(r.Context())
	audit.LogAction(account.ID, "AdminAPI", "ListSchedMessageEvents", nil)
	httputil.JSONResponse(w, http.StatusOK, schedmsg.Events)
}

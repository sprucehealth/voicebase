package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/schedmsg"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type schedMessageEventsListAPIHandler struct{}

func newSchedMessageEventsListAPIHandler() httputil.ContextHandler {
	return httputil.SupportedMethods(&schedMessageEventsListAPIHandler{},
		httputil.Get)
}

func (h *schedMessageEventsListAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "ListSchedMessageEvents", nil)
	httputil.JSONResponse(w, http.StatusOK, schedmsg.Events)
}

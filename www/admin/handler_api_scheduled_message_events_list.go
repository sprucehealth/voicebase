package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/schedmsg"
	"github.com/sprucehealth/backend/www"
)

type schedMessageEventsListAPIHandler struct{}

func newSchedMessageEventsListAPIHandler() httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&schedMessageEventsListAPIHandler{},
		httputil.Get)
}

func (h *schedMessageEventsListAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "ListSchedMessageEvents", nil)
	httputil.JSONResponse(w, http.StatusOK, schedmsg.Events)
}

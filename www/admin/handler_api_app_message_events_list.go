package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/www"
)

type appMessageEventsListAPIHandler struct{}

func NewAppMessageEventsListAPIHandler() http.Handler {
	return httputil.SupportedMethods(&appMessageEventsListAPIHandler{},
		[]string{"GET"})
}

func (h *appMessageEventsListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "ListAppMessageEvents", nil)
	www.JSONResponse(w, r, http.StatusOK, common.ScheduledMessageEvents)
}

package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type syncGlobalFTPHandler struct {
	dataAPI api.DataAPI
}

func newSyncGlobalFTPHandler(
	dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		&syncGlobalFTPHandler{
			dataAPI: dataAPI,
		}, httputil.Post)
}

func (h *syncGlobalFTPHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)

	doctorID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "SyncGlobalFTPs", map[string]interface{}{
		"doctor_id": doctorID,
	})

	if err := h.dataAPI.SyncGlobalFTPsForDoctor(doctorID); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

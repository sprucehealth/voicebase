package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
)

type providerAttributesAPIHandler struct {
	dataAPI api.DataAPI
}

func newProviderAttributesAPIHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&providerAttributesAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get)
}

func (h *providerAttributesAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "GetDoctorAttributes", map[string]interface{}{"doctor_id": doctorID})

	attributes, err := h.dataAPI.DoctorAttributes(doctorID, nil)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, attributes)
}

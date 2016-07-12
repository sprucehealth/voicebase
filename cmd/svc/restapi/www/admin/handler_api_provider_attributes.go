package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
)

type providerAttributesAPIHandler struct {
	dataAPI api.DataAPI
}

func newProviderAttributesAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&providerAttributesAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get)
}

func (h *providerAttributesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(r.Context())["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := www.MustCtxAccount(r.Context())
	audit.LogAction(account.ID, "AdminAPI", "GetDoctorAttributes", map[string]interface{}{"doctor_id": doctorID})

	attributes, err := h.dataAPI.DoctorAttributes(doctorID, nil)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, attributes)
}

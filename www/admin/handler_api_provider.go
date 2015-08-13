package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
)

type providerAPIHandler struct {
	dataAPI api.DataAPI
}

func newProviderAPIHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&providerAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get)
}

func (h *providerAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "GetDoctor", map[string]interface{}{"doctor_id": doctorID})

	doctor, err := h.dataAPI.GetDoctorFromID(doctorID)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Doctor *common.Doctor `json:"doctor"`
	}{
		Doctor: doctor,
	})
}

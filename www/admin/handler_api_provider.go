package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type providerAPIHandler struct {
	dataAPI api.DataAPI
}

type updateProviderRequest struct {
	PrescriberID int64 `json:"prescriber_id,string"`
}

func newProviderAPIHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&providerAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Patch)
}

func (h *providerAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(ctx)

	switch r.Method {
	case httputil.Get:
		audit.LogAction(account.ID, "AdminAPI", "GetDoctor", map[string]interface{}{"doctor_id": doctorID})
	case httputil.Patch:
		audit.LogAction(account.ID, "AdminAPI", "UpdateDoctor", map[string]interface{}{"doctor_id": doctorID})

		var rd updateProviderRequest
		if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
			www.APIInternalError(w, r, err)
			return
		} else if rd.PrescriberID == 0 {
			www.APIBadRequestError(w, r, "prescriber_id required")
			return
		}

		if err := h.dataAPI.UpdateDoctor(doctorID, &api.DoctorUpdate{
			DosespotClinicianID: ptr.Int64(rd.PrescriberID),
		}); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	}

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

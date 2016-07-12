package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/ptr"
)

type providerAPIHandler struct {
	dataAPI api.DataAPI
}

type updateProviderRequest struct {
	PrescriberID int64 `json:"prescriber_id,string"`
}

func newProviderAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&providerAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Patch)
}

func (h *providerAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(r.Context())["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(r.Context())

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

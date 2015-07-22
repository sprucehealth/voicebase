package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type refillRxDenialReasonsHandler struct {
	dataAPI api.DataAPI
}

func NewRefillRxDenialReasonsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&refillRxDenialReasonsHandler{
				dataAPI: dataAPI,
			}), api.RoleDoctor, api.RoleCC),
		httputil.Get)
}

type RefillRequestDenialReasonsResponse struct {
	DenialReasons []*api.RefillRequestDenialReason `json:"refill_request_denial_reasons"`
}

func (d *refillRxDenialReasonsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	denialReasons, err := d.dataAPI.GetRefillRequestDenialReasons()
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &RefillRequestDenialReasonsResponse{DenialReasons: denialReasons})
}

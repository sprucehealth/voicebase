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
	return httputil.SupportedMethods(apiservice.NoAuthorizationRequired(&refillRxDenialReasonsHandler{
		dataAPI: dataAPI,
	}), []string{"GET"})
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
	apiservice.WriteJSON(w, &RefillRequestDenialReasonsResponse{DenialReasons: denialReasons})
}

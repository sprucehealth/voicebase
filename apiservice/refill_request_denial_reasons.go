package apiservice

import (
	"github.com/sprucehealth/backend/api"
	"net/http"
)

type RefillRequestDenialReasonsHandler struct {
	DataApi api.DataAPI
}

type RefillRequestDenialReasonsResponse struct {
	DenialReasons []*api.RefillRequestDenialReason `json:"refill_request_denial_reasons"`
}

func (d *RefillRequestDenialReasonsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	denialReasons, err := d.DataApi.GetRefillRequestDenialReasons()
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get refill request denial reasons: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &RefillRequestDenialReasonsResponse{DenialReasons: denialReasons})
}

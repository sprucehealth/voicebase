package apiservice

import (
	"carefront/api"
	"carefront/common"
	"net/http"

	"github.com/gorilla/schema"
)

type DoctorRefillRequestHandler struct {
	DataApi api.DataAPI
}

type DoctorRefillRequestResponse struct {
	RefillRequest *common.RefillRequestItem `json:"refill_request,omitempty"`
}

type DoctorRefillRequestRequestData struct {
	RefillRequestId int64 `schema:"refill_request_id,required"`
}

func (d *DoctorRefillRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &DoctorRefillRequestRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	refillRequest, err := d.DataApi.GetRefillRequestFromId(requestData.RefillRequestId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get refill request based on id: "+err.Error())
		return
	}

	// fill in the dispense unit description at the top level because it is not provided in the top level
	// information from dosespot
	refillRequest.RequestedDispenseUnitDescription = refillRequest.DispensedPrescription.DispenseUnitDescription

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorRefillRequestResponse{RefillRequest: refillRequest})
}

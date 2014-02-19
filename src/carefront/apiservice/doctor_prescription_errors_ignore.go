package apiservice

import (
	"carefront/api"
	"carefront/libs/erx"
	"net/http"

	"github.com/gorilla/schema"
)

type DoctorPrescriptionErrorIgnoreHandler struct {
	DataApi api.DataAPI
	ErxApi  erx.ERxAPI
}

type DoctorPrescriptionErrorIgnoreRequestData struct {
	PrescriptionId int64 `schema:"erx_id,required"`
}

func (d *DoctorPrescriptionErrorIgnoreHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData DoctorPrescriptionErrorIgnoreRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	if err := d.ErxApi.IgnoreAlert(requestData.PrescriptionId); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to ignore transmission error for prescription: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

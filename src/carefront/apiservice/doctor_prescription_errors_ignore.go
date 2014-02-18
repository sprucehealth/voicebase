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

	switch r.Method {
	case "POST":
	default:
		WriteDeveloperError(w, http.StatusNotFound, "")
		return

	}

	r.ParseForm()
	requestData := new(DoctorPrescriptionErrorIgnoreRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	err = d.ErxApi.IgnoreAlert(requestData.PrescriptionId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to ignore transmission error for prescription: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

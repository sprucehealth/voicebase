package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type DoctorPrescriptionErrorIgnoreHandler struct {
	DataApi api.DataAPI
	ErxApi  erx.ERxAPI
}

type DoctorPrescriptionErrorIgnoreRequestData struct {
	TreatmentId string `schema:"treatment_id,required"`
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

	doctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get doctor from account id: "+err.Error())
		return
	}

	var requestData DoctorPrescriptionErrorIgnoreRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	treatmentId, err := strconv.ParseInt(requestData.TreatmentId, 10, 64)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treament id : "+err.Error())
		return
	}

	treatment, err := d.DataApi.GetTreatmentFromId(treatmentId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment based on id: "+err.Error())
		return
	}

	if err := d.ErxApi.IgnoreAlert(doctor.DoseSpotClinicianId, treatment.PrescriptionId.Int64()); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to ignore transmission error for prescription: "+err.Error())
		return
	}

	if err := d.DataApi.AddErxStatusEvent([]*common.Treatment{treatment}, api.ERX_STATUS_RESOLVED); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add a status of resolved once the error is resolved: "+err.Error())
		return
	}

	if err := d.DataApi.MarkErrorResolvedInDoctorQueue(doctor.DoctorId.Int64(), treatment.Id.Int64(), api.QUEUE_ITEM_STATUS_PENDING, api.QUEUE_ITEM_STATUS_COMPLETED); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to refresh the doctor queue: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

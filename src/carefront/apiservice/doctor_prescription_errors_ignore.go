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
	TreatmentId             string `schema:"treatment_id"`
	RefillRequestId         string `schema:"refill_request_id"`
	UnlinkedDNTFTreatmentId string `schema:"unlinked_dntf_treatment_id"`
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

	if requestData.TreatmentId != "" {
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

		if err := verifyDoctorPatientRelationship(d.DataApi, treatment.Doctor, treatment.Patient); err != nil {
			WriteDeveloperError(w, http.StatusForbidden, "Unable to verify patient-doctor relationship: "+err.Error())
			return
		}

		if err := d.ErxApi.IgnoreAlert(doctor.DoseSpotClinicianId, treatment.ERx.PrescriptionId.Int64()); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to ignore transmission error for prescription: "+err.Error())
			return
		}

		if err := d.DataApi.AddErxStatusEvent([]*common.Treatment{treatment}, common.StatusEvent{Status: api.ERX_STATUS_RESOLVED}); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add a status of resolved once the error is resolved: "+err.Error())
			return
		}

		if err := d.DataApi.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  doctor.DoctorId.Int64(),
			ItemId:    treatment.Id.Int64(),
			EventType: api.EVENT_TYPE_TRANSMISSION_ERROR,
			Status:    api.QUEUE_ITEM_STATUS_COMPLETED,
		}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to refresh the doctor queue: "+err.Error())
			return
		}
	} else if requestData.RefillRequestId != "" {
		refillRequestId, err := strconv.ParseInt(requestData.RefillRequestId, 10, 64)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treament id : "+err.Error())
			return
		}

		refillRequest, err := d.DataApi.GetRefillRequestFromId(refillRequestId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get refill request based on id: "+err.Error())
			return
		}

		if err := verifyDoctorPatientRelationship(d.DataApi, refillRequest.Doctor, refillRequest.Patient); err != nil {
			WriteDeveloperError(w, http.StatusForbidden, "Unable to verify patient-doctor relationship: "+err.Error())
			return
		}

		if err := d.ErxApi.IgnoreAlert(doctor.DoseSpotClinicianId, refillRequest.PrescriptionId); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to ignore transmission error for prescription: "+err.Error())
			return
		}

		if err := d.DataApi.AddRefillRequestStatusEvent(common.StatusEvent{ItemId: refillRequest.Id, Status: api.RX_REFILL_STATUS_ERROR_RESOLVED}); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add a status of resolved once the error is resolved: "+err.Error())
			return
		}

		if err := d.DataApi.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  doctor.DoctorId.Int64(),
			ItemId:    refillRequest.Id,
			EventType: api.EVENT_TYPE_REFILL_TRANSMISSION_ERROR,
			Status:    api.QUEUE_ITEM_STATUS_COMPLETED,
		}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to refresh the doctor queue: "+err.Error())
			return
		}
	} else if requestData.UnlinkedDNTFTreatmentId != "" {
		unlinkedDNTFTreatmentId, err := strconv.ParseInt(requestData.UnlinkedDNTFTreatmentId, 10, 64)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse unlinked dntf treatment id: "+err.Error())
			return
		}

		unlinkedDNTFTreatment, err := d.DataApi.GetUnlinkedDNTFTreatment(unlinkedDNTFTreatmentId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the unlinked dntf treatment: "+err.Error())
			return
		}

		if err := d.ErxApi.IgnoreAlert(doctor.DoseSpotClinicianId, unlinkedDNTFTreatment.ERx.PrescriptionId.Int64()); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to ignore transmission error for prescription: "+err.Error())
			return
		}

		if err := d.DataApi.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
			ItemId: unlinkedDNTFTreatmentId,
			Status: api.ERX_STATUS_RESOLVED,
		}); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to ignore transmission error for unlinked dntf treatment: "+err.Error())
			return
		}

		if err := d.DataApi.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  doctor.DoctorId.Int64(),
			ItemId:    unlinkedDNTFTreatmentId,
			EventType: api.EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR,
			Status:    api.QUEUE_ITEM_STATUS_COMPLETED,
		}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to refresh the doctor queue: "+err.Error())
			return
		}
	} else {
		WriteDeveloperError(w, http.StatusBadRequest, "Require either the treatment id or the refill request id or the unlinked dntf treatment id to ignore a particular error")
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

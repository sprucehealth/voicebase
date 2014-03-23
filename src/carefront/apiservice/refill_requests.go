package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"net/http"
	"sort"

	"github.com/gorilla/schema"
)

const (
	refill_request_status_approve = "approve"
	refill_request_status_deny    = "deny"
)

var (
	actionToRefillRequestStateMapping = map[string]string{
		refill_request_status_approve: api.RX_REFILL_STATUS_APPROVED,
		refill_request_status_deny:    api.RX_REFILL_STATUS_DENIED,
	}
	actionToQueueStateMapping = map[string]string{
		refill_request_status_approve: api.QUEUE_ITEM_STATUS_REFILL_APPROVED,
		refill_request_status_deny:    api.QUEUE_ITEM_STATUS_REFILL_DENIED,
	}
)

type DoctorRefillRequestHandler struct {
	DataApi        api.DataAPI
	ErxApi         erx.ERxAPI
	ErxStatusQueue *common.SQSQueue
}

type DoctorRefillRequestResponse struct {
	RefillRequest *common.RefillRequestItem `json:"refill_request,omitempty"`
}

type DoctorRefillRequestRequestData struct {
	RefillRequestId      int64  `schema:"refill_request_id,required"`
	DenialReasonId       int64  `schema:"denial_reason_id"`
	Comments             string `schema:"comments"`
	Action               string `schema:"action"`
	ApprovedRefillAmount int64  `schema:"approved_refill_amount"`
}

func (d *DoctorRefillRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_GET:
		d.getRefillRequest(w, r)
	case HTTP_PUT:
		d.resolveRefillRequest(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *DoctorRefillRequestHandler) resolveRefillRequest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &DoctorRefillRequestRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	refillRequest, err := d.DataApi.GetRefillRequestFromId(requestData.RefillRequestId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get refill request from id: "+err.Error())
		return
	}

	doctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	if doctor.DoctorId.Int64() != refillRequest.Doctor.DoctorId.Int64() {
		WriteDeveloperError(w, http.StatusBadRequest, "The doctor in the refill request is not the same doctor as the one trying to resolve the request.")
		return
	}

	if len(refillRequest.RxHistory) == 0 {
		WriteDeveloperError(w, http.StatusInternalServerError, "Expected status events for refill requests but none found")
		return
	}

	// Ensure that the refill request is in the Requested state for
	// the user to work on it. If it's in the desired end state, then do nothing
	if refillRequest.RxHistory[0].Status == actionToRefillRequestStateMapping[requestData.Action] {
		// Move the queue item for the doctor from the ongoing to the completed state
		if err := d.DataApi.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  doctor.DoctorId.Int64(),
			ItemId:    refillRequest.Id,
			EventType: api.EVENT_TYPE_REFILL_REQUEST,
			Status:    requestData.Action,
		}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to update the doctor queue with the refill request item")
			return
		}
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
		return
	}

	if refillRequest.RxHistory[0].Status != api.RX_REFILL_STATUS_REQUESTED {
		WriteDeveloperError(w, http.StatusBadRequest, "Cannot approve the refill request for one that is not in the requested state. Current state: "+refillRequest.RxHistory[0].Status)
		return
	}

	switch requestData.Action {
	case refill_request_status_approve:
		// Ensure that the number of refills is non-zero. If its not,
		// report it to the user as a user error
		if requestData.ApprovedRefillAmount == 0 {
			WriteUserError(w, http.StatusBadRequest, "Number of refills to approve has to be greater than 0.")
			return
		}

		// Send the approve refill request to dosespot
		prescriptionId, err := d.ErxApi.ApproveRefillRequest(doctor.DoseSpotClinicianId, refillRequest.RxRequestQueueItemId, requestData.ApprovedRefillAmount, requestData.Comments)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to approve refill request: "+err.Error())
			return
		}

		// Update the refill request entry with the approved refill amount and the returned prescription id
		if err := d.DataApi.MarkRefillRequestAsApproved(prescriptionId, requestData.ApprovedRefillAmount, refillRequest.Id, requestData.Comments); err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to store the updates to the refill request to mark it as being approved: "+err.Error())
			return
		}

	case refill_request_status_deny:

		// Ensure that the denial reason is one of the possible denial reasons that the user could have
		denialReasons, err := d.DataApi.GetRefillRequestDenialReasons()
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the possible reasons for denial for this refill request: "+err.Error())
			return
		}

		var denialReasonCode string
		for _, denialReason := range denialReasons {
			if denialReason.Id == requestData.DenialReasonId {
				denialReasonCode = denialReason.DenialCode
				break
			}
		}

		if denialReasonCode == "" {
			WriteDeveloperError(w, http.StatusBadRequest, "Denial reason code not found based on id specified")
			return
		}

		//  Deny the refill request
		prescriptionId, err := d.ErxApi.DenyRefillRequest(doctor.DoseSpotClinicianId, refillRequest.RxRequestQueueItemId, denialReasonCode, requestData.Comments)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to deny refill request on the dosespot platform for the following reason: "+err.Error())
			return
		}

		//  Update the refill request with the reason for denial and the erxid returned
		if err := d.DataApi.MarkRefillRequestAsDenied(prescriptionId, requestData.DenialReasonId, refillRequest.Id, requestData.Comments); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the refill request to denied: "+err.Error())
			return
		}

	default:
		WriteDeveloperError(w, http.StatusBadRequest, "Expected an action of approve or deny for refill request, instead got "+requestData.Action)
		return
	}

	// Move the queue item for the doctor from the ongoing to the completed state
	if err := d.DataApi.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
		DoctorId:  doctor.DoctorId.Int64(),
		ItemId:    refillRequest.Id,
		EventType: api.EVENT_TYPE_REFILL_REQUEST,
		Status:    actionToQueueStateMapping[requestData.Action],
	}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to update the doctor queue with the refill request item")
		return
	}

	//  Queue up job to check for whether or not the response to this refill request
	// was successfully transmitted to the pharmacy
	if err := queueUpJobForErxStatus(d.ErxStatusQueue, common.PrescriptionStatusCheckMessage{
		PatientId:          refillRequest.Patient.PatientId.Int64(),
		DoctorId:           refillRequest.Doctor.DoctorId.Int64(),
		CheckRefillRequest: true,
	}); err != nil {
		golog.Errorf("Unable to enqueue job into sqs queue to keep track of refill request status. Not erroring out to user because there's nothing they can do about it: %+v", err)
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

func (d *DoctorRefillRequestHandler) getRefillRequest(w http.ResponseWriter, r *http.Request) {
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

	if refillRequest != nil {
		// fill in the dispense unit description at the top level because the dispense unit description	 is not provided in the top level
		// information from dosespot
		refillRequest.RequestedDispenseUnitDescription = refillRequest.DispensedPrescription.DispenseUnitDescription
	}

	if refillRequest.RequestedPrescription.OriginatingTreatmentId != 0 {
		rxHistoryOfOriginatingTreatment, err := d.DataApi.GetPrescriptionStatusEventsForTreatment(refillRequest.RequestedPrescription.OriginatingTreatmentId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get rxhistory of the originating treatment: "+err.Error())
			return
		}

		// add these events to the rx history of the refill request
		refillRequest.RxHistory = append(refillRequest.RxHistory, rxHistoryOfOriginatingTreatment...)
		sort.Reverse(common.ByStatusTimestamp(refillRequest.RxHistory))
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorRefillRequestResponse{RefillRequest: refillRequest})
}

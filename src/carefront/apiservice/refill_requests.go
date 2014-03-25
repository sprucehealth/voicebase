package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"encoding/json"
	"net/http"
	"sort"
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
	RefillRequestId      *common.ObjectId  `json:"refill_request_id,required"`
	DenialReasonId       *common.ObjectId  `json:"denial_reason_id"`
	Comments             string            `json:"comments"`
	Action               string            `json:"action"`
	ApprovedRefillAmount int64             `json:"approved_refill_amount"`
	Treatment            *common.Treatment `json:"new_treatment,omitempty"`
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

	requestData := &DoctorRefillRequestRequestData{}
	if err := json.NewDecoder(r.Body).Decode(requestData); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
	}

	refillRequest, err := d.DataApi.GetRefillRequestFromId(requestData.RefillRequestId.Int64())
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
			if denialReason.Id == requestData.DenialReasonId.Int64() {
				denialReasonCode = denialReason.DenialCode
				break
			}
		}

		if denialReasonCode == "" {
			WriteDeveloperError(w, http.StatusBadRequest, "Denial reason code not found based on id specified")
			return
		}

		// if denial reason is DNTF then make sure that there is a treatment along with the denial request
		if denialReasonCode == "DeniedNewRx" {

			if requestData.Treatment == nil {
				WriteDeveloperErrorWithCode(w, DEVELOPER_TREATMENT_MISSING_DNTF, http.StatusBadRequest, "Treatment missing when reason for denial selected as denied new request to follow.")
				return
			}

			// validate the treatment
			if err := validateTreatment(requestData.Treatment); err != nil {
				WriteUserError(w, http.StatusBadRequest, err.Error())
				return
			}

			// break up the name in its components
			drugName, drugForm, drugRoute := breakDrugInternalNameIntoComponents(requestData.Treatment.DrugInternalName)
			requestData.Treatment.DrugName = drugName
			requestData.Treatment.DrugForm = drugForm
			requestData.Treatment.DrugRoute = drugRoute

			httpStatusCode, errorResponse := checkIfDrugInTreatmentFromTemplateIsOutOfMarket(requestData.Treatment, doctor, d.ErxApi)
			if errorResponse != nil {
				WriteError(w, httpStatusCode, *errorResponse)
				return
			}

			if refillRequest.ReferenceNumber == "" {
				WriteDeveloperError(w, http.StatusInternalServerError, "Cannot proceed with refill request denial as reference number for refill request is missing which is required to deny with new request to follow")
				return
			}

			// assign the reference number to the treatment so that when it is added it is linked to the refill request
			if requestData.Treatment.ERx == nil {
				requestData.Treatment.ERx = &common.ERxData{}
			}
			requestData.Treatment.ERx.ErxReferenceNumber = refillRequest.ReferenceNumber

			// add the treatment for the patient
			if err := d.DataApi.AddTreatmentInEventOfDNTF(requestData.Treatment, doctor.DoctorId.Int64(), refillRequest.Patient.PatientId.Int64(), refillRequest.Id); err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment in event of DNTF: "+err.Error())
				return
			}

			//  start prescribing
			if err := d.ErxApi.StartPrescribingPatient(doctor.DoseSpotClinicianId, refillRequest.Patient, []*common.Treatment{requestData.Treatment}); err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to start prescribing to get back prescription id for treatment: "+err.Error())
				return
			}

			// save prescription id for drug to database
			if err := d.DataApi.UpdateTreatmentWithPharmacyAndErxId([]*common.Treatment{requestData.Treatment}, refillRequest.RequestedPrescription.ERx.Pharmacy, doctor.DoctorId.Int64()); err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update treatment with erx id and pharmacy information for treatment: "+err.Error())
				return
			}

			//  send prescription to pharmacy
			unSuccesfulTreatmentIds, err := d.ErxApi.SendMultiplePrescriptions(doctor.DoseSpotClinicianId, refillRequest.Patient, []*common.Treatment{requestData.Treatment})
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to send prescription to pharmacy: "+err.Error())
				return
			}

			// ensure its successful
			for _, unSuccessfulTreatmentId := range unSuccesfulTreatmentIds {
				if unSuccessfulTreatmentId == requestData.Treatment.Id.Int64() {
					if err := d.DataApi.AddErxStatusEvent([]*common.Treatment{requestData.Treatment}, common.StatusEvent{Status: api.ERX_STATUS_SEND_ERROR}); err != nil {
						WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add an erx status event: "+err.Error())
						return
					}
					WriteDeveloperError(w, http.StatusInternalServerError, "Unable to send prescription to pharmacy: "+err.Error())
					return
				}
			}

			if err := d.DataApi.AddErxStatusEvent([]*common.Treatment{requestData.Treatment}, common.StatusEvent{Status: api.ERX_STATUS_SENT}); err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add status event for treatment: "+err.Error())
				return
			}

			// queue up job for status checking
			if err := queueUpJobForErxStatus(d.ErxStatusQueue, common.PrescriptionStatusCheckMessage{
				PatientId: refillRequest.Patient.PatientId.Int64(),
				DoctorId:  doctor.DoctorId.Int64(),
			}); err != nil {
				golog.Errorf("Unable to enqueue job to check status of erx for new rx after DNTF. Not going to error out on this for the user becuase there is nothing the user can do about this: %+v", err)
			}
		}

		//  Deny the refill request
		prescriptionId, err := d.ErxApi.DenyRefillRequest(doctor.DoseSpotClinicianId, refillRequest.RxRequestQueueItemId, denialReasonCode, requestData.Comments)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to deny refill request on the dosespot platform for the following reason: "+err.Error())
			return
		}

		//  Update the refill request with the reason for denial and the erxid returned
		if err := d.DataApi.MarkRefillRequestAsDenied(prescriptionId, requestData.DenialReasonId.Int64(), refillRequest.Id, requestData.Comments); err != nil {
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
	requestData := &DoctorRefillRequestRequestData{}
	if err := json.NewDecoder(r.Body).Decode(requestData); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
	}

	refillRequest, err := d.DataApi.GetRefillRequestFromId(requestData.RefillRequestId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get refill request based on id: "+err.Error())
		return
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

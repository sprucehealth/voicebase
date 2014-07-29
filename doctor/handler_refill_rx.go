package doctor

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/surescripts"
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
		refill_request_status_approve: api.DQItemStatusRefillApproved,
		refill_request_status_deny:    api.DQItemStatusRefillDenied,
	}
)

type refillRxHandler struct {
	dataAPI        api.DataAPI
	erxAPI         erx.ERxAPI
	erxStatusQueue *common.SQSQueue
}

func NewRefillRxHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI, erxStatusQueue *common.SQSQueue) http.Handler {
	return &refillRxHandler{
		dataAPI:        dataAPI,
		erxAPI:         erxAPI,
		erxStatusQueue: erxStatusQueue,
	}
}

type DoctorRefillRequestResponse struct {
	RefillRequest *common.RefillRequestItem `json:"refill_request,omitempty"`
}

type DoctorRefillRequestRequestData struct {
	RefillRequestId      encoding.ObjectId `json:"refill_request_id,required"`
	DenialReasonId       encoding.ObjectId `json:"denial_reason_id"`
	Comments             string            `json:"comments"`
	Action               string            `json:"action"`
	ApprovedRefillAmount int64             `json:"approved_refill_amount"`
	Treatment            *common.Treatment `json:"new_treatment,omitempty"`
}

type DoctorGetRefillRequestData struct {
	RefillRequestId int64 `schema:"refill_request_id"`
}

func (d *refillRxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		d.getRefillRequest(w, r)
	case apiservice.HTTP_PUT:
		d.resolveRefillRequest(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *refillRxHandler) resolveRefillRequest(w http.ResponseWriter, r *http.Request) {
	requestData := &DoctorRefillRequestRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	refillRequest, err := d.dataAPI.GetRefillRequestFromId(requestData.RefillRequestId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if len(requestData.Comments) > surescripts.MaxRefillRequestCommentLength {
		apiservice.WriteValidationError("Comments for refill request cannot be greater than 70 characters", w, r)
		return
	}

	doctor, err := d.dataAPI.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if doctor.DoctorId.Int64() != refillRequest.Doctor.DoctorId.Int64() {
		apiservice.WriteValidationError("The doctor in the refill request is not the same doctor as the one trying to resolve the request.", w, r)
		return
	}

	if len(refillRequest.RxHistory) == 0 {
		apiservice.WriteError(err, w, r)
		return
	}
	// Ensure that the refill request is in the Requested state for
	// the user to work on it. If it's in the desired end state, then do nothing
	if refillRequest.RxHistory[0].Status == actionToRefillRequestStateMapping[requestData.Action] {
		dispatch.Default.Publish(&RefillRequestResolvedEvent{
			DoctorId:        doctor.DoctorId.Int64(),
			Status:          actionToRefillRequestStateMapping[requestData.Action],
			RefillRequestId: refillRequest.Id,
		})
		apiservice.WriteJSONSuccess(w)
		return
	}

	if refillRequest.RxHistory[0].Status != api.RX_REFILL_STATUS_REQUESTED {
		apiservice.WriteValidationError("Cannot approve the refill request for one that is not in the requested state. Current state: "+refillRequest.RxHistory[0].Status, w, r)
		return
	}

	switch requestData.Action {
	case refill_request_status_approve:
		// Ensure that the number of refills is non-zero. If its not,
		// report it to the user as a user error
		if requestData.ApprovedRefillAmount == 0 {
			apiservice.WriteValidationError("Number of refills to approve has to be greater than 0", w, r)
			return
		}

		trimSpacesFromRefillRequest(refillRequest)

		// get the refill request to check if it is a controlled substance
		refillRequest, err := d.DataApi.GetRefillRequestFromId(requestData.RefillRequestId.Int64())
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get refill request from id: "+err.Error())
			return
		}

		// we cannot let the doctor approve this refill request given that we are dealing with
		// a controlled substance
		if refillRequest.RequestedPrescription.IsControlledSubstance {
			WriteUserError(w, StatusUnprocessableEntity, "Unfortunately, we do not support electronic routing of controlled substances using the platform. The only option available is to deny the refill request. If you have any questions, feel free to contact support. Apologies for any inconvenience!")
			return
		}

		// Send the approve refill request to dosespot
		prescriptionId, err := d.erxAPI.ApproveRefillRequest(doctor.DoseSpotClinicianId, refillRequest.RxRequestQueueItemId, requestData.ApprovedRefillAmount, requestData.Comments)
		if err != nil {
			apiservice.WriteValidationError("Unable to approve refill request: "+err.Error(), w, r)
			return
		}

		// Update the refill request entry with the approved refill amount and the returned prescription id
		if err := d.dataAPI.MarkRefillRequestAsApproved(prescriptionId, requestData.ApprovedRefillAmount, refillRequest.Id, requestData.Comments); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

	case refill_request_status_deny:

		trimSpacesFromRefillRequest(refillRequest)

		// Ensure that the denial reason is one of the possible denial reasons that the user could have
		denialReasons, err := d.dataAPI.GetRefillRequestDenialReasons()
		if err != nil {
			apiservice.WriteError(err, w, r)
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
			apiservice.WriteValidationError("Denial reason code not found based on id specified", w, r)
			return
		}

		// if denial reason is DNTF then make sure that there is a treatment along with the denial request
		if denialReasonCode == api.RX_REFILL_DNTF_REASON_CODE {

			if requestData.Treatment == nil {
				apiservice.WriteDeveloperErrorWithCode(w, apiservice.DEVELOPER_TREATMENT_MISSING_DNTF, http.StatusBadRequest, "Treatment missing when reason for denial selected as denied new request to follow.")
				return
			}

			// validate the treatment
			if err := apiservice.ValidateTreatment(requestData.Treatment); err != nil {
				apiservice.WriteValidationError(err.Error(), w, r)
				return
			}

			apiservice.TrimSpacesFromTreatmentFields(requestData.Treatment)

			// break up the name in its components
			requestData.Treatment.DrugName, requestData.Treatment.DrugForm, requestData.Treatment.DrugRoute = apiservice.BreakDrugInternalNameIntoComponents(requestData.Treatment.DrugInternalName)

			httpStatusCode, errorResponse := apiservice.CheckIfDrugInTreatmentFromTemplateIsOutOfMarket(requestData.Treatment, doctor, d.erxAPI)
			if errorResponse != nil {
				apiservice.WriteErrorResponse(w, httpStatusCode, *errorResponse)
				return
			}

			if refillRequest.ReferenceNumber == "" {
				apiservice.WriteValidationError("Cannot proceed with refill request denial as reference number for refill request is missing which is required to deny with new request to follow", w, r)
				return
			}

			// assign the reference number to the treatment so that when it is added it is linked to the refill request
			if requestData.Treatment.ERx == nil {
				requestData.Treatment.ERx = &common.ERxData{}
			}
			// NOTE: we are required to send in the RxRequestQueueItemId according to DoseSpot
			requestData.Treatment.ERx.ErxReferenceNumber = strconv.FormatInt(refillRequest.RxRequestQueueItemId, 10)
			originatingTreatmentFound := refillRequest.RequestedPrescription.OriginatingTreatmentId != 0

			if originatingTreatmentFound {
				originatingTreatment, err := d.dataAPI.GetTreatmentFromId(refillRequest.RequestedPrescription.OriginatingTreatmentId)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				requestData.Treatment.TreatmentPlanId = originatingTreatment.TreatmentPlanId
			}

			//  Deny the refill request
			prescriptionId, err := d.ErxApi.DenyRefillRequest(doctor.DoseSpotClinicianId, refillRequest.RxRequestQueueItemId, denialReasonCode, requestData.Comments)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to deny refill request on the dosespot platform for the following reason: "+err.Error())
				return
			} else if err := d.DataApi.MarkRefillRequestAsDenied(prescriptionId, requestData.DenialReasonId.Int64(), refillRequest.Id, requestData.Comments); err != nil {
				//  Update the refill request with the reason for denial and the erxid returned
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the refill request to denied: "+err.Error())
				return
			}

			if err := d.addTreatmentInEventOfDNTF(originatingTreatmentFound, requestData.Treatment, refillRequest.Doctor.DoctorId.Int64(), refillRequest.Patient.PatientId.Int64(), refillRequest.Id); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			//  start prescribing
			if err := d.erxAPI.StartPrescribingPatient(doctor.DoseSpotClinicianId, refillRequest.Patient, []*common.Treatment{requestData.Treatment}, refillRequest.RequestedPrescription.ERx.Pharmacy.SourceId); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			// update pharmacy and erx id for treatment
			if err := d.updateTreatmentWithPharmacyAndErxId(originatingTreatmentFound, requestData.Treatment, refillRequest.RequestedPrescription.ERx.Pharmacy, doctor.DoctorId.Int64()); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			//  send prescription to pharmacy
			unSuccesfulTreatments, err := d.ErxApi.SendMultiplePrescriptions(doctor.DoseSpotClinicianId, refillRequest.Patient, []*common.Treatment{requestData.Treatment})
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			// ensure its successful
			for _, unSuccessfulTreatment := range unSuccesfulTreatments {
				if unSuccessfulTreatment.Id.Int64() == requestData.Treatment.Id.Int64() {
					if err := d.addStatusEvent(originatingTreatmentFound, requestData.Treatment, common.StatusEvent{Status: api.ERX_STATUS_SEND_ERROR}); err != nil {
						apiservice.WriteError(err, w, r)
						return
					}
					apiservice.WriteError(err, w, r)
					return
				}
			}

			if err := d.addStatusEvent(originatingTreatmentFound, requestData.Treatment, common.StatusEvent{ItemId: requestData.Treatment.Id.Int64(), Status: api.ERX_STATUS_SENDING}); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			eventCheckType := common.ERxType
			if !originatingTreatmentFound {
				eventCheckType = common.UnlinkedDNTFTreatmentType
			}

			// queue up job for status checking
			if err := apiservice.QueueUpJobForErxStatus(d.erxStatusQueue, common.PrescriptionStatusCheckMessage{
				PatientId:      refillRequest.Patient.PatientId.Int64(),
				DoctorId:       doctor.DoctorId.Int64(),
				EventCheckType: eventCheckType,
			}); err != nil {
				golog.Errorf("Unable to enqueue job to check status of erx for new rx after DNTF. Not going to error out on this for the user becuase there is nothing the user can do about this: %+v", err)
			}
		} else {
			//  Deny the refill request
			prescriptionId, err := d.ErxApi.DenyRefillRequest(doctor.DoseSpotClinicianId, refillRequest.RxRequestQueueItemId, denialReasonCode, requestData.Comments)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to deny refill request on the dosespot platform for the following reason: "+err.Error())
				return
			} else if err := d.DataApi.MarkRefillRequestAsDenied(prescriptionId, requestData.DenialReasonId.Int64(), refillRequest.Id, requestData.Comments); err != nil {
				//  Update the refill request with the reason for denial and the erxid returned
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the refill request to denied: "+err.Error())
				return
			}
		}

	default:
		apiservice.WriteValidationError("Expected an action of approve or deny for refill request, instead got "+requestData.Action, w, r)
		return
	}

	dispatch.Default.Publish(&RefillRequestResolvedEvent{
		DoctorId:        doctor.DoctorId.Int64(),
		Status:          actionToQueueStateMapping[requestData.Action],
		RefillRequestId: refillRequest.Id,
	})

	//  Queue up job to check for whether or not the response to this refill request
	// was successfully transmitted to the pharmacy
	if err := apiservice.QueueUpJobForErxStatus(d.erxStatusQueue, common.PrescriptionStatusCheckMessage{
		PatientId:      refillRequest.Patient.PatientId.Int64(),
		DoctorId:       refillRequest.Doctor.DoctorId.Int64(),
		EventCheckType: common.RefillRxType,
	}); err != nil {
		golog.Errorf("Unable to enqueue job into sqs queue to keep track of refill request status. Not erroring out to user because there's nothing they can do about it: %+v", err)
	}

	apiservice.WriteJSONSuccess(w)
}

func (d *refillRxHandler) updateTreatmentWithPharmacyAndErxId(originatingTreatmentFound bool, treatment *common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorId int64) error {
	if originatingTreatmentFound {
		return d.dataAPI.UpdateTreatmentWithPharmacyAndErxId([]*common.Treatment{treatment}, pharmacySentTo, doctorId)
	}
	return d.dataAPI.UpdateUnlinkedDNTFTreatmentWithPharmacyAndErxId(treatment, pharmacySentTo, doctorId)
}

func (d *refillRxHandler) addStatusEvent(originatingTreatmentFound bool, treatment *common.Treatment, statusEvent common.StatusEvent) error {
	if originatingTreatmentFound {
		return d.DataApi.AddErxStatusEvent([]*common.Treatment{treatment}, statusEvent)
	}
	return d.dataAPI.AddErxStatusEventForDNTFTreatment(statusEvent)
}

func (d *refillRxHandler) addTreatmentInEventOfDNTF(originatingTreatmentFound bool, treatment *common.Treatment, doctorId, patientId, refillRequestId int64) error {
	treatment.PatientId = encoding.NewObjectId(patientId)
	treatment.DoctorId = encoding.NewObjectId(doctorId)
	if originatingTreatmentFound {
		return d.dataAPI.AddTreatmentToTreatmentPlanInEventOfDNTF(treatment, refillRequestId)
	}
	return d.dataAPI.AddUnlinkedTreatmentInEventOfDNTF(treatment, refillRequestId)
}

func (d *refillRxHandler) getRefillRequest(w http.ResponseWriter, r *http.Request) {
	requestData := &DoctorGetRefillRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	refillRequest, err := d.dataAPI.GetRefillRequestFromId(requestData.RefillRequestId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if refillRequest.RequestedPrescription.OriginatingTreatmentId != 0 {
		rxHistoryOfOriginatingTreatment, err := d.dataAPI.GetPrescriptionStatusEventsForTreatment(refillRequest.RequestedPrescription.OriginatingTreatmentId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// add these events to the rx history of the refill request
		refillRequest.RxHistory = append(refillRequest.RxHistory, rxHistoryOfOriginatingTreatment...)
		sort.Reverse(common.ByStatusTimestamp(refillRequest.RxHistory))
	}
	apiservice.WriteJSON(w, &DoctorRefillRequestResponse{RefillRequest: refillRequest})
}

func trimSpacesFromRefillRequest(refillRequest *common.RefillRequestItem) {
	refillRequest.Comments = strings.TrimSpace(refillRequest.Comments)
}

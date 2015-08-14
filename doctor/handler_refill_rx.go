package doctor

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/surescripts"
)

const (
	refillRequestStatusApprove = "approve"
	refillRequestStatusDeny    = "deny"
)

var (
	actionToRefillRequestStateMapping = map[string]string{
		refillRequestStatusApprove: api.RXRefillStatusApproved,
		refillRequestStatusDeny:    api.RXRefillStatusDenied,
	}
	actionToQueueStateMapping = map[string]string{
		refillRequestStatusApprove: api.DQItemStatusRefillApproved,
		refillRequestStatusDeny:    api.DQItemStatusRefillDenied,
	}
)

type refillRxHandler struct {
	dataAPI        api.DataAPI
	erxAPI         erx.ERxAPI
	dispatcher     *dispatch.Dispatcher
	erxStatusQueue *common.SQSQueue
}

func NewRefillRxHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI, dispatcher *dispatch.Dispatcher, erxStatusQueue *common.SQSQueue) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&refillRxHandler{
				dataAPI:        dataAPI,
				erxAPI:         erxAPI,
				dispatcher:     dispatcher,
				erxStatusQueue: erxStatusQueue,
			})),
		httputil.Get, httputil.Put)
}

type DoctorRefillRequestResponse struct {
	RefillRequest *common.RefillRequestItem `json:"refill_request,omitempty"`
}

type DoctorRefillRequestRequestData struct {
	RefillRequestID      int64             `json:"refill_request_id,string" schema:"refill_request_id,required"`
	DenialReasonID       int64             `json:"denial_reason_id,string"`
	Comments             string            `json:"comments"`
	Action               string            `json:"action"`
	ApprovedRefillAmount int64             `json:"approved_refill_amount"`
	Treatment            *common.Treatment `json:"new_treatment,omitempty"`
}

func (d *refillRxHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		d.getRefillRequest(ctx, w, r)
	case httputil.Put:
		d.resolveRefillRequest(ctx, w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *refillRxHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	requestData := &DoctorRefillRequestRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = requestData

	refillRequest, err := d.dataAPI.GetRefillRequestFromID(requestData.RefillRequestID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKRefillRequest] = refillRequest

	doctor, err := d.dataAPI.GetDoctorFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctor] = doctor

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, account.Role, doctor.ID.Int64(), refillRequest.Patient.ID, d.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (d *refillRxHandler) resolveRefillRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	requestData := requestCache[apiservice.CKRequestData].(*DoctorRefillRequestRequestData)
	doctor := requestCache[apiservice.CKDoctor].(*common.Doctor)
	refillRequest := requestCache[apiservice.CKRefillRequest].(*common.RefillRequestItem)

	if len(requestData.Comments) > surescripts.MaxRefillRequestCommentLength {
		apiservice.WriteValidationError(ctx, "Comments for refill request cannot be greater than 70 characters", w, r)
		return
	}

	if doctor.ID.Int64() != refillRequest.Doctor.ID.Int64() {
		apiservice.WriteValidationError(ctx, "The doctor in the refill request is not the same doctor as the one trying to resolve the request.", w, r)
		return
	}

	// Ensure that the refill request is in the Requested state for
	// the user to work on it. If it's in the desired end state, then do nothing
	if refillRequest.RxHistory[0].Status == actionToRefillRequestStateMapping[requestData.Action] {
		d.dispatcher.Publish(&RefillRequestResolvedEvent{
			Doctor:          doctor,
			Status:          actionToRefillRequestStateMapping[requestData.Action],
			RefillRequestID: refillRequest.ID,
		})
		apiservice.WriteJSONSuccess(w)
		return
	}

	if refillRequest.RxHistory[0].Status != api.RXRefillStatusRequested {
		apiservice.WriteValidationError(ctx, "Cannot approve the refill request for one that is not in the requested state. Current state: "+refillRequest.RxHistory[0].Status, w, r)
		return
	}

	switch requestData.Action {
	case refillRequestStatusApprove:
		// Ensure that the number of refills is non-zero. If its not,
		// report it to the user as a user error
		if requestData.ApprovedRefillAmount == 0 {
			apiservice.WriteValidationError(ctx, "Number of refills to approve has to be greater than 0", w, r)
			return
		}

		trimSpacesFromRefillRequest(refillRequest)

		// get the refill request to check if it is a controlled substance
		refillRequest, err := d.dataAPI.GetRefillRequestFromID(requestData.RefillRequestID)
		if err != nil {
			apiservice.WriteError(ctx, fmt.Errorf("Unable to get refill request from ID %d: %s", requestData.RefillRequestID, err), w, r)
			return
		}

		// we cannot let the doctor approve this refill request given that we are dealing with
		// a controlled substance
		if refillRequest.RequestedPrescription.IsControlledSubstance {
			apiservice.WriteUserError(w, apiservice.StatusUnprocessableEntity, "Unfortunately, we do not support electronic routing of controlled substances using the platform. The only option available is to deny the refill request. If you have any questions, feel free to contact support. Apologies for any inconvenience!")
			return
		}

		// Send the approve refill request to dosespot
		prescriptionID, err := d.erxAPI.ApproveRefillRequest(doctor.DoseSpotClinicianID, refillRequest.RxRequestQueueItemID, requestData.ApprovedRefillAmount, requestData.Comments)
		if err != nil {
			apiservice.WriteValidationError(ctx, "Unable to approve refill request: "+err.Error(), w, r)
			return
		}

		// Update the refill request entry with the approved refill amount and the returned prescription id
		if err := d.dataAPI.MarkRefillRequestAsApproved(prescriptionID, requestData.ApprovedRefillAmount, refillRequest.ID, requestData.Comments); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

	case refillRequestStatusDeny:
		trimSpacesFromRefillRequest(refillRequest)

		// Ensure that the denial reason is one of the possible denial reasons that the user could have
		denialReasons, err := d.dataAPI.GetRefillRequestDenialReasons()
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		var denialReasonCode string
		for _, denialReason := range denialReasons {
			if denialReason.ID == requestData.DenialReasonID {
				denialReasonCode = denialReason.DenialCode
				break
			}
		}

		if denialReasonCode == "" {
			apiservice.WriteValidationError(ctx, "Denial reason code not found based on id specified", w, r)
			return
		}

		// if denial reason is DNTF then make sure that there is a treatment along with the denial request
		if denialReasonCode == api.RXRefillDNTFReasonCode {
			if requestData.Treatment == nil {
				apiservice.WriteDeveloperErrorWithCode(w, apiservice.DeveloperErrorTreatmentMissingDNTF, http.StatusBadRequest, "Treatment missing when reason for denial selected as denied new request to follow.")
				return
			}

			// validate the treatment
			if err := apiservice.ValidateTreatment(requestData.Treatment); err != nil {
				apiservice.WriteValidationError(ctx, err.Error(), w, r)
				return
			}

			apiservice.TrimSpacesFromTreatmentFields(requestData.Treatment)

			// break up the name in its components
			requestData.Treatment.DrugName, requestData.Treatment.DrugForm, requestData.Treatment.DrugRoute = apiservice.BreakDrugInternalNameIntoComponents(requestData.Treatment.DrugInternalName)

			if requestData.Treatment.DoctorTreatmentTemplateID.Int64() != 0 {
				if err := apiservice.IsDrugOutOfMarket(requestData.Treatment, doctor, d.erxAPI); err != nil {
					apiservice.WriteError(ctx, err, w, r)
					return
				}
			}

			if refillRequest.ReferenceNumber == "" {
				apiservice.WriteValidationError(ctx, "Cannot proceed with refill request denial as reference number for refill request is missing which is required to deny with new request to follow", w, r)
				return
			}

			// assign the reference number to the treatment so that when it is added it is linked to the refill request
			if requestData.Treatment.ERx == nil {
				requestData.Treatment.ERx = &common.ERxData{}
			}
			// NOTE: we are required to send in the RxRequestQueueItemId according to DoseSpot
			requestData.Treatment.ERx.ErxReferenceNumber = strconv.FormatInt(refillRequest.RxRequestQueueItemID, 10)
			originatingTreatmentFound := refillRequest.RequestedPrescription.OriginatingTreatmentID != 0

			if originatingTreatmentFound {
				originatingTreatment, err := d.dataAPI.GetTreatmentFromID(refillRequest.RequestedPrescription.OriginatingTreatmentID)
				if err != nil {
					apiservice.WriteError(ctx, err, w, r)
					return
				}
				requestData.Treatment.TreatmentPlanID = originatingTreatment.TreatmentPlanID
			}

			//  Deny the refill request
			prescriptionID, err := d.erxAPI.DenyRefillRequest(doctor.DoseSpotClinicianID, refillRequest.RxRequestQueueItemID, denialReasonCode, requestData.Comments)
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			} else if err := d.dataAPI.MarkRefillRequestAsDenied(prescriptionID, requestData.DenialReasonID, refillRequest.ID, requestData.Comments); err != nil {
				//  Update the refill request with the reason for denial and the erxid returned
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			if err := d.addTreatmentInEventOfDNTF(originatingTreatmentFound, requestData.Treatment, refillRequest.Doctor.ID.Int64(), refillRequest.Patient.ID, refillRequest.ID); err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			//  start prescribing
			if err := d.erxAPI.StartPrescribingPatient(doctor.DoseSpotClinicianID, refillRequest.Patient, []*common.Treatment{requestData.Treatment}, refillRequest.RequestedPrescription.ERx.Pharmacy.SourceID); err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			// update pharmacy and erx id for treatment
			if err := d.updateTreatmentWithPharmacyAndERxID(originatingTreatmentFound, requestData.Treatment, refillRequest.RequestedPrescription.ERx.Pharmacy, doctor.ID.Int64()); err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			//  send prescription to pharmacy
			unSuccesfulTreatments, err := d.erxAPI.SendMultiplePrescriptions(doctor.DoseSpotClinicianID, refillRequest.Patient, []*common.Treatment{requestData.Treatment})
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			// ensure its successful
			for _, unSuccessfulTreatment := range unSuccesfulTreatments {
				if unSuccessfulTreatment.ID.Int64() == requestData.Treatment.ID.Int64() {
					if err := d.addStatusEvent(originatingTreatmentFound, requestData.Treatment, common.StatusEvent{Status: api.ERXStatusSendError}); err != nil {
						apiservice.WriteError(ctx, err, w, r)
						return
					}
					apiservice.WriteError(ctx, err, w, r)
					return
				}
			}

			if err := d.addStatusEvent(originatingTreatmentFound, requestData.Treatment, common.StatusEvent{ItemID: requestData.Treatment.ID.Int64(), Status: api.ERXStatusSending}); err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			eventCheckType := common.ERxType
			if !originatingTreatmentFound {
				eventCheckType = common.UnlinkedDNTFTreatmentType
			}

			// queue up job for status checking
			if err := apiservice.QueueUpJob(d.erxStatusQueue, &common.PrescriptionStatusCheckMessage{
				PatientID:      refillRequest.Patient.ID,
				DoctorID:       doctor.ID.Int64(),
				EventCheckType: eventCheckType,
			}); err != nil {
				golog.Errorf("Unable to enqueue job to check status of erx for new rx after DNTF. Not going to error out on this for the user becuase there is nothing the user can do about this: %+v", err)
			}
		} else {
			//  Deny the refill request
			prescriptionID, err := d.erxAPI.DenyRefillRequest(doctor.DoseSpotClinicianID, refillRequest.RxRequestQueueItemID, denialReasonCode, requestData.Comments)
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			} else if err := d.dataAPI.MarkRefillRequestAsDenied(prescriptionID, requestData.DenialReasonID, refillRequest.ID, requestData.Comments); err != nil {
				//  Update the refill request with the reason for denial and the erxid returned
				apiservice.WriteError(ctx, err, w, r)
				return
			}
		}

	default:
		apiservice.WriteValidationError(ctx, "Expected an action of approve or deny for refill request, instead got "+requestData.Action, w, r)
		return
	}

	d.dispatcher.Publish(&RefillRequestResolvedEvent{
		Patient:         refillRequest.Patient,
		Doctor:          doctor,
		Status:          actionToQueueStateMapping[requestData.Action],
		RefillRequestID: refillRequest.ID,
	})

	// Queue up job to check for whether or not the response to this refill request
	// was successfully transmitted to the pharmacy
	if err := apiservice.QueueUpJob(d.erxStatusQueue, &common.PrescriptionStatusCheckMessage{
		PatientID:      refillRequest.Patient.ID,
		DoctorID:       refillRequest.Doctor.ID.Int64(),
		EventCheckType: common.RefillRxType,
	}); err != nil {
		golog.Errorf("Unable to enqueue job into sqs queue to keep track of refill request status. Not erroring out to user because there's nothing they can do about it: %+v", err)
	}

	apiservice.WriteJSONSuccess(w)
}

func (d *refillRxHandler) updateTreatmentWithPharmacyAndERxID(originatingTreatmentFound bool, treatment *common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorID int64) error {
	if originatingTreatmentFound {
		return d.dataAPI.UpdateTreatmentWithPharmacyAndErxID([]*common.Treatment{treatment}, pharmacySentTo, doctorID)
	}
	return d.dataAPI.UpdateUnlinkedDNTFTreatmentWithPharmacyAndErxID(treatment, pharmacySentTo, doctorID)
}

func (d *refillRxHandler) addStatusEvent(originatingTreatmentFound bool, treatment *common.Treatment, statusEvent common.StatusEvent) error {
	if originatingTreatmentFound {
		return d.dataAPI.AddErxStatusEvent([]*common.Treatment{treatment}, statusEvent)
	}
	return d.dataAPI.AddErxStatusEventForDNTFTreatment(statusEvent)
}

func (d *refillRxHandler) addTreatmentInEventOfDNTF(originatingTreatmentFound bool, treatment *common.Treatment, doctorID int64, patientID common.PatientID, refillRequestID int64) error {
	treatment.PatientID = patientID
	treatment.DoctorID = encoding.DeprecatedNewObjectID(doctorID)
	if originatingTreatmentFound {
		return d.dataAPI.AddTreatmentToTreatmentPlanInEventOfDNTF(treatment, refillRequestID)
	}
	return d.dataAPI.AddUnlinkedTreatmentInEventOfDNTF(treatment, refillRequestID)
}

func (d *refillRxHandler) getRefillRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	refillRequest := requestCache[apiservice.CKRefillRequest].(*common.RefillRequestItem)

	if refillRequest.RequestedPrescription.OriginatingTreatmentID != 0 {
		rxHistoryOfOriginatingTreatment, err := d.dataAPI.GetPrescriptionStatusEventsForTreatment(refillRequest.RequestedPrescription.OriginatingTreatmentID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		// add these events to the rx history of the refill request
		refillRequest.RxHistory = append(refillRequest.RxHistory, rxHistoryOfOriginatingTreatment...)
		sort.Sort(sort.Reverse(common.ByStatusTimestamp(refillRequest.RxHistory)))
	}
	httputil.JSONResponse(w, http.StatusOK, &DoctorRefillRequestResponse{RefillRequest: refillRequest})
}

func trimSpacesFromRefillRequest(refillRequest *common.RefillRequestItem) {
	refillRequest.Comments = strings.TrimSpace(refillRequest.Comments)
}

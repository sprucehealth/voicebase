package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
)

type prescriptionErrorIgnoreHandler struct {
	dataAPI    api.DataAPI
	erxAPI     erx.ERxAPI
	dispatcher *dispatch.Dispatcher
}

func NewPrescriptionErrorIgnoreHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return apiservice.AuthorizationRequired(&prescriptionErrorIgnoreHandler{
		dataAPI:    dataAPI,
		erxAPI:     erxAPI,
		dispatcher: dispatcher,
	})
}

type DoctorPrescriptionErrorIgnoreRequestData struct {
	TreatmentID             int64 `schema:"treatment_id"`
	RefillRequestID         int64 `schema:"refill_request_id"`
	UnlinkedDNTFTreatmentId int64 `schema:"unlinked_dntf_treatment_id"`
}

func (d *prescriptionErrorIgnoreHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	ctxt := apiservice.GetContext(r)

	var requestData DoctorPrescriptionErrorIgnoreRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = &requestData

	if requestData.TreatmentID != 0 {
		treatment, err := d.dataAPI.GetTreatmentFromID(requestData.TreatmentID)
		if err != nil {
			return false, err
		}

		if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, treatment.Doctor.DoctorID.Int64(),
			treatment.Patient.PatientID.Int64(), d.dataAPI); err != nil {
			return false, err
		}

		ctxt.RequestCache[apiservice.Treatment] = treatment
		ctxt.RequestCache[apiservice.ERxSource] = common.ERxType
	} else if requestData.RefillRequestID != 0 {
		refillRequest, err := d.dataAPI.GetRefillRequestFromID(requestData.RefillRequestID)
		if err != nil {
			return false, err
		}

		if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, refillRequest.Doctor.DoctorID.Int64(),
			refillRequest.Patient.PatientID.Int64(), d.dataAPI); err != nil {
			return false, err
		}

		ctxt.RequestCache[apiservice.RefillRequest] = refillRequest
		ctxt.RequestCache[apiservice.ERxSource] = common.RefillRxType
	} else if requestData.UnlinkedDNTFTreatmentId != 0 {
		unlinkedDNTFTreatment, err := d.dataAPI.GetUnlinkedDNTFTreatment(requestData.UnlinkedDNTFTreatmentId)
		if err != nil {
			return false, err
		}

		ctxt.RequestCache[apiservice.Treatment] = unlinkedDNTFTreatment
		ctxt.RequestCache[apiservice.ERxSource] = common.UnlinkedDNTFTreatmentType
	}

	return true, nil
}

func (d *prescriptionErrorIgnoreHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	doctor, err := d.dataAPI.GetDoctorFromAccountID(ctxt.AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var itemID int64
	var patient *common.Patient

	eventType := apiservice.GetContext(r).RequestCache[apiservice.ERxSource].(common.ERxSourceType)
	switch eventType {
	case common.ERxType:
		treatment := ctxt.RequestCache[apiservice.Treatment].(*common.Treatment)
		patient = treatment.Patient
		if err := d.erxAPI.IgnoreAlert(doctor.DoseSpotClinicianID, treatment.ERx.PrescriptionID.Int64()); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.AddErxStatusEvent([]*common.Treatment{treatment}, common.StatusEvent{Status: api.ERX_STATUS_RESOLVED}); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		itemID = treatment.ID.Int64()
	case common.UnlinkedDNTFTreatmentType:
		unlinkedDNTFTreatment := ctxt.RequestCache[apiservice.Treatment].(*common.Treatment)
		patient = unlinkedDNTFTreatment.Patient
		if err := d.erxAPI.IgnoreAlert(doctor.DoseSpotClinicianID, unlinkedDNTFTreatment.ERx.PrescriptionID.Int64()); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
			ItemID: unlinkedDNTFTreatment.ID.Int64(),
			Status: api.ERX_STATUS_RESOLVED,
		}); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		itemID = unlinkedDNTFTreatment.ID.Int64()
	case common.RefillRxType:
		refillRequest := ctxt.RequestCache[apiservice.RefillRequest].(*common.RefillRequestItem)
		patient = refillRequest.Patient

		if err := d.erxAPI.IgnoreAlert(doctor.DoseSpotClinicianID, refillRequest.PrescriptionID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.AddRefillRequestStatusEvent(common.StatusEvent{ItemID: refillRequest.ID, Status: api.RX_REFILL_STATUS_ERROR_RESOLVED}); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		itemID = refillRequest.ID
	default:
		apiservice.WriteValidationError("Either treatment_Id, refill_request_id or unlinked_dntf_treatment_id must be specified", w, r)
		return
	}

	d.dispatcher.Publish(&RxTransmissionErrorResolvedEvent{
		Doctor:    doctor,
		ItemID:    itemID,
		EventType: eventType,
		Patient:   patient,
	})

	apiservice.WriteJSONSuccess(w)
}

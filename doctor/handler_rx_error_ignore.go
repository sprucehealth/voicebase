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
	TreatmentId             int64 `schema:"treatment_id"`
	RefillRequestId         int64 `schema:"refill_request_id"`
	UnlinkedDNTFTreatmentId int64 `schema:"unlinked_dntf_treatment_id"`
}

func (d *prescriptionErrorIgnoreHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	ctxt := apiservice.GetContext(r)

	var requestData DoctorPrescriptionErrorIgnoreRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = &requestData

	if requestData.TreatmentId != 0 {
		treatment, err := d.dataAPI.GetTreatmentFromId(requestData.TreatmentId)
		if err != nil {
			return false, err
		}

		if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, treatment.Doctor.DoctorId.Int64(),
			treatment.Patient.PatientId.Int64(), d.dataAPI); err != nil {
			return false, err
		}

		ctxt.RequestCache[apiservice.Treatment] = treatment
		ctxt.RequestCache[apiservice.ERxSource] = common.ERxType
	} else if requestData.RefillRequestId != 0 {
		refillRequest, err := d.dataAPI.GetRefillRequestFromId(requestData.RefillRequestId)
		if err != nil {
			return false, err
		}

		if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, refillRequest.Doctor.DoctorId.Int64(),
			refillRequest.Patient.PatientId.Int64(), d.dataAPI); err != nil {
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
	doctor, err := d.dataAPI.GetDoctorFromAccountId(ctxt.AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var itemId int64

	eventType := apiservice.GetContext(r).RequestCache[apiservice.ERxSource].(common.ERxSourceType)
	switch eventType {
	case common.ERxType:
		treatment := ctxt.RequestCache[apiservice.Treatment].(*common.Treatment)

		if err := d.erxAPI.IgnoreAlert(doctor.DoseSpotClinicianId, treatment.ERx.PrescriptionId.Int64()); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.AddErxStatusEvent([]*common.Treatment{treatment}, common.StatusEvent{Status: api.ERX_STATUS_RESOLVED}); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		itemId = treatment.Id.Int64()
	case common.UnlinkedDNTFTreatmentType:
		unlinkedDNTFTreatment := ctxt.RequestCache[apiservice.Treatment].(*common.Treatment)
		if err := d.erxAPI.IgnoreAlert(doctor.DoseSpotClinicianId, unlinkedDNTFTreatment.ERx.PrescriptionId.Int64()); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
			ItemId: unlinkedDNTFTreatment.Id.Int64(),
			Status: api.ERX_STATUS_RESOLVED,
		}); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		itemId = unlinkedDNTFTreatment.Id.Int64()
	case common.RefillRxType:
		refillRequest := ctxt.RequestCache[apiservice.RefillRequest].(*common.RefillRequestItem)

		if err := d.erxAPI.IgnoreAlert(doctor.DoseSpotClinicianId, refillRequest.PrescriptionId); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.AddRefillRequestStatusEvent(common.StatusEvent{ItemId: refillRequest.Id, Status: api.RX_REFILL_STATUS_ERROR_RESOLVED}); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		itemId = refillRequest.Id
	default:
		apiservice.WriteValidationError("Either treatment_Id, refill_request_id or unlinked_dntf_treatment_id must be specified", w, r)
		return
	}

	d.dispatcher.Publish(&RxTransmissionErrorResolvedEvent{
		DoctorId:  doctor.DoctorId.Int64(),
		ItemId:    itemId,
		EventType: eventType,
	})

	apiservice.WriteJSONSuccess(w)
}

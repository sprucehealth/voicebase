package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
)

type doctorPrescriptionErrorIgnoreHandler struct {
	dataAPI api.DataAPI
	erxAPI  erx.ERxAPI
}

func NewDoctorPrescriptionErrorIgnoreHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI) http.Handler {
	return &doctorPrescriptionErrorIgnoreHandler{
		dataAPI: dataAPI,
		erxAPI:  erxAPI,
	}
}

type DoctorPrescriptionErrorIgnoreRequestData struct {
	TreatmentId             int64 `schema:"treatment_id"`
	RefillRequestId         int64 `schema:"refill_request_id"`
	UnlinkedDNTFTreatmentId int64 `schema:"unlinked_dntf_treatment_id"`
}

func (d *doctorPrescriptionErrorIgnoreHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := GetContext(r)

	var requestData DoctorPrescriptionErrorIgnoreRequestData
	if err := DecodeRequestData(&requestData, r); err != nil {
		return false, NewValidationError(err.Error(), r)
	} else {
		ctxt.RequestCache[RequestData] = &requestData
	}

	if requestData.TreatmentId != 0 {
		treatment, err := d.dataAPI.GetTreatmentFromId(requestData.TreatmentId)
		if err != nil {
			return false, err
		}

		if !treatment.Patient.IsUnlinked {
			if err := ValidateDoctorAccessToPatientFile(treatment.Doctor.DoctorId.Int64(), treatment.Patient.PatientId.Int64(), d.dataAPI); err != nil {
				return false, err
			}
		}

		ctxt.RequestCache[Treatment] = treatment
		ctxt.RequestCache[ERxSource] = common.ERxType
	} else if requestData.RefillRequestId != 0 {
		refillRequest, err := d.dataAPI.GetRefillRequestFromId(requestData.RefillRequestId)
		if err != nil {
			return false, err
		}

		if !refillRequest.Patient.IsUnlinked {
			if err := ValidateDoctorAccessToPatientFile(refillRequest.Doctor.DoctorId.Int64(), refillRequest.Patient.PatientId.Int64(), d.dataAPI); err != nil {
				return false, err
			}
		}

		ctxt.RequestCache[RefillRequest] = refillRequest
		ctxt.RequestCache[ERxSource] = common.RefillRxType
	} else if requestData.UnlinkedDNTFTreatmentId != 0 {
		unlinkedDNTFTreatment, err := d.dataAPI.GetUnlinkedDNTFTreatment(requestData.UnlinkedDNTFTreatmentId)
		if err != nil {
			return false, err
		}

		ctxt.RequestCache[Treatment] = unlinkedDNTFTreatment
		ctxt.RequestCache[ERxSource] = common.UnlinkedDNTFTreatmentType
	}

	return true, nil
}

func (d *doctorPrescriptionErrorIgnoreHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_POST {
		http.NotFound(w, r)
		return
	}

	ctxt := GetContext(r)
	doctor, err := d.dataAPI.GetDoctorFromAccountId(ctxt.AccountId)
	if err != nil {
		WriteError(err, w, r)
		return
	}

	var itemId int64

	eventType := GetContext(r).RequestCache[ERxSource].(common.ERxSourceType)
	switch eventType {
	case common.ERxType:
		treatment := ctxt.RequestCache[Treatment].(*common.Treatment)

		if err := d.erxAPI.IgnoreAlert(doctor.DoseSpotClinicianId, treatment.ERx.PrescriptionId.Int64()); err != nil {
			WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.AddErxStatusEvent([]int64{treatment.Id.Int64()}, common.StatusEvent{Status: api.ERX_STATUS_RESOLVED}); err != nil {
			WriteError(err, w, r)
			return
		}

		itemId = treatment.Id.Int64()
	case common.UnlinkedDNTFTreatmentType:
		unlinkedDNTFTreatment := ctxt.RequestCache[Treatment].(*common.Treatment)
		if err := d.erxAPI.IgnoreAlert(doctor.DoseSpotClinicianId, unlinkedDNTFTreatment.ERx.PrescriptionId.Int64()); err != nil {
			WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
			ItemId: unlinkedDNTFTreatment.Id.Int64(),
			Status: api.ERX_STATUS_RESOLVED,
		}); err != nil {
			WriteError(err, w, r)
			return
		}
		itemId = unlinkedDNTFTreatment.Id.Int64()
	case common.RefillRxType:
		refillRequest := ctxt.RequestCache[RefillRequest].(*common.RefillRequestItem)

		if err := d.erxAPI.IgnoreAlert(doctor.DoseSpotClinicianId, refillRequest.PrescriptionId); err != nil {
			WriteError(err, w, r)
			return
		}

		if err := d.dataAPI.AddRefillRequestStatusEvent(common.StatusEvent{ItemId: refillRequest.Id, Status: api.RX_REFILL_STATUS_ERROR_RESOLVED}); err != nil {
			WriteError(err, w, r)
			return
		}

		itemId = refillRequest.Id
	default:
		WriteValidationError("Either treatment_Id, refill_request_id or unlinked_dntf_treatment_id must be specified", w, r)
		return
	}

	dispatch.Default.Publish(&RxTransmissionErrorResolvedEvent{
		DoctorId:  doctor.DoctorId.Int64(),
		ItemId:    itemId,
		EventType: eventType,
	})

	WriteJSONSuccess(w)
}

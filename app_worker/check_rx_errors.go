package app_worker

import (
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

const (
	waitTimeInMinsForRxErrorChecker = 2 * time.Hour
)

// StartWorkerToCheckRxErrors runs periodically to check for any uncaught erx transmission errors
// for doctors on our platform. This can happen for reasons like:
// a) we forget/fail to enqueue a message to sqs to check status of sqs messages
// b) sqs is down but we want to continue letting doctors route prescritpions
// c) there is an error in sending a prescription after it is registered as being sent to the pharmacy
// d) something else we have not thought of! This is our fallback mechanism to catch all errors
func StartWorkerToCheckRxErrors(dataApi api.DataAPI, erxApi erx.ERxAPI, statsRegistry metrics.Registry) {
	statFailure := metrics.NewCounter()
	statCycles := metrics.NewCounter()

	statsRegistry.Add("cycles/total", statCycles)
	statsRegistry.Add("cycles/failed", statFailure)

	go func() {
		for {
			PerformRxErrorCheck(dataApi, erxApi, statFailure, statCycles)
			statCycles.Inc(1)
			time.Sleep(waitTimeInMinsForRxErrorChecker)
		}
	}()
}

func PerformRxErrorCheck(dataApi api.DataAPI, erxApi erx.ERxAPI, statFailure, statCycles *metrics.Counter) {

	// Get all doctors on our platform
	doctors, err := dataApi.GetAllDoctorsInClinic()
	if err != nil {
		golog.Errorf("Unable to get all doctors in clinic %s", err)
		statFailure.Inc(1)
		return
	}

	for _, doctor := range doctors {

		// nothing to do if doctor does not have a dosespot clinician id
		if doctor.DoseSpotClinicianId == 0 {
			continue
		}

		// get transmission error details for each doctor
		treatmentsWithErrors, err := erxApi.GetTransmissionErrorDetails(doctor.DoseSpotClinicianId)
		if err != nil {
			golog.Errorf("Unable to get transmission error details for doctor id %d. Error : %s", doctor.DoseSpotClinicianId, err)
			statFailure.Inc(1)
			continue
		}

		// nothing to do for this doctor if there are no errors
		if len(treatmentsWithErrors) == 0 {
			continue
		}

		// go through each error and compare the status of the treatment it links to in our database
		for _, treatmentWithError := range treatmentsWithErrors {
			treatment, err := dataApi.GetTreatmentBasedOnPrescriptionId(treatmentWithError.ERx.PrescriptionId.Int64())
			switch err {
			case nil:
				if err := handleErxErrorForTreatmentInTreatmentPlan(dataApi, treatment, treatmentWithError); err != nil {
					statFailure.Inc(1)
				}
				continue
			case api.NoRowsError:
				// prescription not found as a treatment within a treatment plan. Check other places
				// for the existence of the prescription
			default:
				golog.Errorf("Unable to get treatment based on prescription id %d. error: %s", treatmentWithError.ERx.PrescriptionId.Int64(), err)
			}

			refillRequest, err := dataApi.GetRefillRequestFromPrescriptionId(treatmentWithError.ERx.PrescriptionId.Int64())
			switch err {
			case nil:
				if err := handlErxErrorForRefillRequest(dataApi, refillRequest, treatmentWithError); err != nil {
					statFailure.Inc(1)
				}
				continue
			case api.NoRowsError:
				// prescription not found as a refill request. Check unlinked dntf treatment
				// for existence of prescription
			default:
				golog.Errorf(("Unable to get refill request based on prescription id %d. error: %s"), treatmentWithError.ERx.PrescriptionId.Int64(), err)
			}

			unlinkedDNTFTreatment, err := dataApi.GetUnlinkedDNTFTreatmentFromPrescriptionId(treatmentWithError.ERx.PrescriptionId.Int64())
			switch err {
			case nil:
				if err := handlErxErrorForUnlinkedDNTFTreatment(dataApi, unlinkedDNTFTreatment, treatmentWithError); err != nil {
					statFailure.Inc(1)
				}
				continue
			case api.NoRowsError:
				// prescription not found as a treatment within a treatment plan,
				// a refill request or a dntf treatment.

				// TODO its possible (although a rare case) for the prescription to not exist in our system
				// in which case we still have to show the transmission error to the doctor. We will have to create
				// some mechanism to "park" these errors in the database for the doctor
				golog.Debugf("Prescription id %d not found in our database...Ignoring for now.", treatmentWithError.ERx.PrescriptionId.Int64())
				statFailure.Inc(1)
			default:
				golog.Errorf("Error trying to get unlinked dntf treatment based on prescription id %d. error :%s", treatmentWithError.ERx.PrescriptionId.Int64(), err)
				statFailure.Inc(1)
			}
		}
	}
}

func handlErxErrorForUnlinkedDNTFTreatment(dataApi api.DataAPI, unlinkedDNTFTreatment, treatmentWithError *common.Treatment) error {
	statusEvents, err := dataApi.GetErxStatusEventsForDNTFTreatment(unlinkedDNTFTreatment.Id.Int64())
	if err != nil {
		golog.Errorf("Unable to get status events for unlinked dntf treatment id %d. error : %s", unlinkedDNTFTreatment.Id.Int64(), err)
		return err
	}

	// if the latest item does not represent an error, insert
	// an error into the rx history of the unlinked dntf treatment and add a
	// refil request transmission error to the doctor's queue
	if statusEvents[0].Status != api.ERX_STATUS_ERROR {
		if err := dataApi.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
			Status:            api.ERX_STATUS_ERROR,
			StatusDetails:     treatmentWithError.StatusDetails,
			ReportedTimestamp: *treatmentWithError.ERx.TransmissionErrorDate,
			ItemId:            unlinkedDNTFTreatment.Id.Int64(),
		}); err != nil {
			golog.Errorf("Unable to add error event to rx history for unlinked dntf treatment: %s", err.Error())
			return err
		}

		if err := dataApi.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  unlinkedDNTFTreatment.Doctor.DoctorId.Int64(),
			ItemId:    unlinkedDNTFTreatment.Id.Int64(),
			Status:    api.DQItemStatusPending,
			EventType: api.DQEventTypeUnlinkedDNTFTransmissionError,
		}); err != nil {
			golog.Errorf("Unable to insert unlinked dntf treatment transmission error into doctor queue: %s", err)
			return err
		}
	}

	return nil
}

func handlErxErrorForRefillRequest(dataApi api.DataAPI, refillRequest *common.RefillRequestItem, treatmentWithError *common.Treatment) error {
	statusEvents, err := dataApi.GetRefillStatusEventsForRefillRequest(refillRequest.Id)
	if err != nil {
		golog.Errorf("Unable to get status events for refill request id %d. error : %s", refillRequest.Id, err)
		return err
	}

	// if the latest item does not represent an error, insert
	// an error into the rx history of the refill request and add a
	// refil request transmission error to the doctor's queue
	if statusEvents[0].Status != api.RX_REFILL_STATUS_ERROR {
		if err := dataApi.AddRefillRequestStatusEvent(common.StatusEvent{
			Status:            api.RX_REFILL_STATUS_ERROR,
			StatusDetails:     treatmentWithError.StatusDetails,
			ReportedTimestamp: *treatmentWithError.ERx.TransmissionErrorDate,
			ItemId:            refillRequest.Id,
		}); err != nil {
			golog.Errorf("Unable to add error event to rx history for refill request: %s", err.Error())
			return err
		}

		if err := dataApi.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  refillRequest.Doctor.DoctorId.Int64(),
			ItemId:    refillRequest.Id,
			Status:    api.DQItemStatusPending,
			EventType: api.DQEventTypeRefillTransmissionError,
		}); err != nil {
			golog.Errorf("Unable to insert refill transmission error into doctor queue: %+v", err)
			return err
		}
	}

	return nil
}

func handleErxErrorForTreatmentInTreatmentPlan(dataApi api.DataAPI, treatment, treatmentWithError *common.Treatment) error {
	statusEvents, err := dataApi.GetPrescriptionStatusEventsForTreatment(treatment.Id.Int64())
	if err != nil {
		golog.Errorf("Unable to get status events for treatment id %d that was found to have transmission errors: %s", treatment.Id.Int64(), err)
		return err
	}

	// if the latest status item does not represent an error
	// insert an error into the rx history of this treatment and add a
	// transmission error for the doctor
	if len(statusEvents) == 0 || statusEvents[0].Status != api.ERX_STATUS_ERROR {
		if err := dataApi.AddErxStatusEvent([]*common.Treatment{treatment}, common.StatusEvent{
			Status:            api.ERX_STATUS_ERROR,
			StatusDetails:     treatmentWithError.StatusDetails,
			ReportedTimestamp: *treatmentWithError.ERx.TransmissionErrorDate,
			ItemId:            treatment.Id.Int64(),
		}); err != nil {
			golog.Errorf("Unable to add error event for status: %s", err.Error())
			return err
		}

		if err := dataApi.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  treatment.Doctor.DoctorId.Int64(),
			ItemId:    treatment.Id.Int64(),
			Status:    api.DQItemStatusPending,
			EventType: api.DQEventTypeTransmissionError,
		}); err != nil {
			golog.Errorf("Unable to insert refill transmission error into doctor queue: %+v", err)
			return err
		}
	}
	return nil
}

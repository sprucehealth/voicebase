package app_worker

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	waitTimeForRXErrorWorker = 2 * time.Hour
)

type ERxWorker struct {
	dataAPI     api.DataAPI
	erxAPI      erx.ERxAPI
	lockAPI     api.LockAPI
	stopChan    chan bool
	statFailure *metrics.Counter
	statCycles  *metrics.Counter
}

// StartWorkerToCheckRxErrors runs periodically to check for any uncaught erx transmission errors
// for doctors on our platform. This can happen for reasons like:
// a) we forget/fail to enqueue a message to sqs to check status of sqs messages
// b) sqs is down but we want to continue letting doctors route prescritpions
// c) there is an error in sending a prescription after it is registered as being sent to the pharmacy
// d) something else we have not thought of! This is our fallback mechanism to catch all errors
func NewERxErrorWorker(
	dataAPI api.DataAPI,
	erxAPI erx.ERxAPI,
	lockAPI api.LockAPI,
	statsRegistry metrics.Registry) *ERxWorker {
	statFailure := metrics.NewCounter()
	statCycles := metrics.NewCounter()

	statsRegistry.Add("cycles/total", statCycles)
	statsRegistry.Add("cycles/failed", statFailure)

	return &ERxWorker{
		dataAPI:     dataAPI,
		erxAPI:      erxAPI,
		lockAPI:     lockAPI,
		stopChan:    make(chan bool),
		statFailure: statFailure,
		statCycles:  statCycles,
	}
}

func (w *ERxWorker) Start() {
	go func() {
		defer w.lockAPI.Release()
		for {
			if !w.lockAPI.Wait() {
				return
			}

			select {
			case <-w.stopChan:
				return
			default:
			}

			w.Do()
			w.statCycles.Inc(1)

			select {
			case <-w.stopChan:
				return
			case <-time.After(waitTimeForRXErrorWorker):
			}

		}
	}()
}

func (w *ERxWorker) Do() error {
	// Get all doctors on our platform
	doctors, err := w.dataAPI.ListCareProviders(api.LCPOptDoctorsOnly)
	if err != nil {
		golog.Errorf("Unable to get all doctors in clinic: %s", err)
		w.statFailure.Inc(1)
		return err
	}

	for _, doctor := range doctors {
		// nothing to do if doctor does not have a dosespot clinician id
		if doctor.DoseSpotClinicianID == 0 {
			continue
		}

		// get transmission error details for each doctor
		treatmentsWithErrors, err := w.erxAPI.GetTransmissionErrorDetails(doctor.DoseSpotClinicianID)
		if err != nil {
			golog.Errorf("Unable to get transmission error details for doctor id %d. Error : %s", doctor.DoseSpotClinicianID, err)
			w.statFailure.Inc(1)
			continue
		}

		// nothing to do for this doctor if there are no errors
		if len(treatmentsWithErrors) == 0 {
			continue
		}

		// go through each error and compare the status of the treatment it links to in our database
		for _, treatmentWithError := range treatmentsWithErrors {
			treatment, err := w.dataAPI.GetTreatmentBasedOnPrescriptionID(treatmentWithError.ERx.PrescriptionID.Int64())
			if err == nil {
				if err := handleErxErrorForTreatmentInTreatmentPlan(w.dataAPI, treatment, treatmentWithError); err != nil {
					w.statFailure.Inc(1)
				}
				continue
			} else if !api.IsErrNotFound(err) {
				golog.Errorf("Unable to get treatment based on prescription id %d. error: %s", treatmentWithError.ERx.PrescriptionID.Int64(), err)
			}

			// prescription not found as a treatment within a treatment plan. Check other places
			// for the existence of the prescription

			refillRequest, err := w.dataAPI.GetRefillRequestFromPrescriptionID(treatmentWithError.ERx.PrescriptionID.Int64())
			if err == nil {
				if err := handlErxErrorForRefillRequest(w.dataAPI, refillRequest, treatmentWithError); err != nil {
					w.statFailure.Inc(1)
				}
				continue
			} else if !api.IsErrNotFound(err) {
				golog.Errorf(("Unable to get refill request based on prescription id %d. error: %s"), treatmentWithError.ERx.PrescriptionID.Int64(), err)
			}

			// prescription not found as a refill request. Check unlinked dntf treatment
			// for existence of prescription

			unlinkedDNTFTreatment, err := w.dataAPI.GetUnlinkedDNTFTreatmentFromPrescriptionID(treatmentWithError.ERx.PrescriptionID.Int64())
			if err == nil {
				if err := handlErxErrorForUnlinkedDNTFTreatment(w.dataAPI, unlinkedDNTFTreatment, treatmentWithError); err != nil {
					w.statFailure.Inc(1)
				}
				continue
			} else if api.IsErrNotFound(err) {
				// prescription not found as a treatment within a treatment plan,
				// a refill request or a dntf treatment.

				// TODO its possible (although a rare case) for the prescription to not exist in our system
				// in which case we still have to show the transmission error to the doctor. We will have to create
				// some mechanism to "park" these errors in the database for the doctor
				golog.Debugf("Prescription id %d not found in our database...Ignoring for now.", treatmentWithError.ERx.PrescriptionID.Int64())
				w.statFailure.Inc(1)
			} else {
				golog.Errorf("Error trying to get unlinked dntf treatment based on prescription id %d. error :%s", treatmentWithError.ERx.PrescriptionID.Int64(), err)
				w.statFailure.Inc(1)
			}
		}
	}
	return nil
}

func handlErxErrorForUnlinkedDNTFTreatment(dataAPI api.DataAPI, unlinkedDNTFTreatment, treatmentWithError *common.Treatment) error {
	statusEvents, err := dataAPI.GetErxStatusEventsForDNTFTreatment(unlinkedDNTFTreatment.ID.Int64())
	if err != nil {
		golog.Errorf("Unable to get status events for unlinked dntf treatment id %d. error : %s", unlinkedDNTFTreatment.ID.Int64(), err)
		return err
	}

	// if the latest item does not represent an error, insert
	// an error into the rx history of the unlinked dntf treatment and add a
	// refil request transmission error to the doctor's queue
	if statusEvents[0].Status != api.ERXStatusError {
		if err := dataAPI.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
			Status:            api.ERXStatusError,
			StatusDetails:     treatmentWithError.StatusDetails,
			ReportedTimestamp: *treatmentWithError.ERx.TransmissionErrorDate,
			ItemID:            unlinkedDNTFTreatment.ID.Int64(),
		}); err != nil {
			golog.Errorf("Unable to add error event to rx history for unlinked dntf treatment: %s", err.Error())
			return err
		}

		if err := dataAPI.UpdateDoctorQueue([]*api.DoctorQueueUpdate{
			{
				Action: api.DQActionInsert,
				QueueItem: &api.DoctorQueueItem{
					DoctorID:         unlinkedDNTFTreatment.Doctor.ID.Int64(),
					PatientID:        unlinkedDNTFTreatment.Patient.ID.Int64(),
					ItemID:           unlinkedDNTFTreatment.ID.Int64(),
					Status:           api.DQItemStatusPending,
					EventType:        api.DQEventTypeUnlinkedDNTFTransmissionError,
					Description:      fmt.Sprintf("Error sending prescription for %s %s", unlinkedDNTFTreatment.Patient.FirstName, unlinkedDNTFTreatment.Patient.LastName),
					ShortDescription: "Prescription error",
					ActionURL:        app_url.ViewDNTFTransmissionErrorAction(unlinkedDNTFTreatment.Patient.ID.Int64(), unlinkedDNTFTreatment.ID.Int64()),
				},
			},
		}); err != nil {
			golog.Errorf("Unable to insert unlinked dntf treatment transmission error into doctor queue: %s", err)
			return err
		}
	}

	return nil
}

func handlErxErrorForRefillRequest(dataAPI api.DataAPI, refillRequest *common.RefillRequestItem, treatmentWithError *common.Treatment) error {
	statusEvents, err := dataAPI.GetRefillStatusEventsForRefillRequest(refillRequest.ID)
	if err != nil {
		golog.Errorf("Unable to get status events for refill request id %d. error : %s", refillRequest.ID, err)
		return err
	}

	// don't insert an error into the doctor queue if there is an event that indicates the error
	// was already resolved or accounted for
	for _, event := range statusEvents {
		switch event.Status {
		case api.RXRefillStatusErrorResolved, api.RXRefillStatusError:
			return nil
		}
	}

	if err := dataAPI.AddRefillRequestStatusEvent(common.StatusEvent{
		Status:            api.RXRefillStatusError,
		StatusDetails:     treatmentWithError.StatusDetails,
		ReportedTimestamp: *treatmentWithError.ERx.TransmissionErrorDate,
		ItemID:            refillRequest.ID,
	}); err != nil {
		golog.Errorf("Unable to add error event to rx history for refill request: %s", err.Error())
		return err
	}

	if err := dataAPI.UpdateDoctorQueue([]*api.DoctorQueueUpdate{
		{
			Action: api.DQActionInsert,
			QueueItem: &api.DoctorQueueItem{
				DoctorID:         refillRequest.Doctor.ID.Int64(),
				PatientID:        refillRequest.Patient.ID.Int64(),
				ItemID:           refillRequest.ID,
				Status:           api.DQItemStatusPending,
				EventType:        api.DQEventTypeRefillTransmissionError,
				Description:      fmt.Sprintf("Error completing refill request for %s %s", refillRequest.Patient.FirstName, refillRequest.Patient.LastName),
				ShortDescription: "Refill request error",
				ActionURL:        app_url.ViewRefillRequestAction(refillRequest.Patient.ID.Int64(), refillRequest.ID),
			},
		},
	}); err != nil {
		golog.Errorf("Unable to insert refill transmission error into doctor queue: %+v", err)
		return err
	}

	return nil
}

func handleErxErrorForTreatmentInTreatmentPlan(dataAPI api.DataAPI, treatment, treatmentWithError *common.Treatment) error {
	statusEvents, err := dataAPI.GetPrescriptionStatusEventsForTreatment(treatment.ID.Int64())
	if err != nil {
		golog.Errorf("Unable to get status events for treatment id %d that was found to have transmission errors: %s", treatment.ID.Int64(), err)
		return err
	}

	// don't insert an error into the doctor queue if there is an event
	// that indicates that the error was already resolved or that the error
	// has already been reported.
	for _, event := range statusEvents {
		switch event.Status {
		case api.ERXStatusResolved, api.ERXStatusError:
			return nil
		}
	}

	if err := dataAPI.AddErxStatusEvent([]*common.Treatment{treatment}, common.StatusEvent{
		Status:            api.ERXStatusError,
		StatusDetails:     treatmentWithError.StatusDetails,
		ReportedTimestamp: *treatmentWithError.ERx.TransmissionErrorDate,
		ItemID:            treatment.ID.Int64(),
	}); err != nil {
		golog.Errorf("Unable to add error event for status: %s", err.Error())
		return err
	}

	if err := dataAPI.UpdateDoctorQueue([]*api.DoctorQueueUpdate{
		{
			Action: api.DQActionInsert,
			QueueItem: &api.DoctorQueueItem{
				DoctorID:         treatment.Doctor.ID.Int64(),
				PatientID:        treatment.Patient.ID.Int64(),
				ItemID:           treatment.ID.Int64(),
				Status:           api.DQItemStatusPending,
				EventType:        api.DQEventTypeTransmissionError,
				Description:      fmt.Sprintf("Error sending prescription for %s %s", treatment.Patient.FirstName, treatment.Patient.LastName),
				ShortDescription: "Prescription error",
				ActionURL:        app_url.ViewTransmissionErrorAction(treatment.Patient.ID.Int64(), treatment.ID.Int64()),
			},
		},
	}); err != nil {
		golog.Errorf("Unable to insert refill transmission error into doctor queue: %+v", err)
		return err
	}
	return nil
}

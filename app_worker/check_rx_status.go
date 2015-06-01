package app_worker

import (
	"encoding/json"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"

	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
)

const (
	waitTimeInSeconds     = 30
	msgVisibilityTimeout  = 30
	longPollingTimePeriod = 20
)

type ERxStatusWorker struct {
	dataAPI         api.DataAPI
	erxAPI          erx.ERxAPI
	dispatcher      *dispatch.Dispatcher
	erxQueue        *common.SQSQueue
	statProcessTime metrics.Histogram
	statCycles      *metrics.Counter
	statFailure     *metrics.Counter
	stopChan        chan bool
}

func NewERxStatusWorker(
	dataAPI api.DataAPI,
	erxAPI erx.ERxAPI,
	dispatcher *dispatch.Dispatcher,
	erxQueue *common.SQSQueue,
	statsRegistry metrics.Registry) *ERxStatusWorker {

	statProcessTime := metrics.NewBiasedHistogram()
	statCycles := metrics.NewCounter()
	statFailure := metrics.NewCounter()

	statsRegistry.Add("cycles/total", statCycles)
	statsRegistry.Add("cycles/processTime", statProcessTime)
	statsRegistry.Add("cycles/failed", statFailure)

	return &ERxStatusWorker{
		dataAPI:         dataAPI,
		erxAPI:          erxAPI,
		dispatcher:      dispatcher,
		erxQueue:        erxQueue,
		statProcessTime: statProcessTime,
		statCycles:      statCycles,
		statFailure:     statFailure,
		stopChan:        make(chan bool),
	}
}

func (w *ERxStatusWorker) Start() {
	go func() {
		for {

			select {
			case <-w.stopChan:
				return
			default:
			}

			if err := w.Do(); err != nil {
				golog.Errorf(err.Error())
			}

			select {
			case <-w.stopChan:
				return
			case <-time.After(waitTimeInSeconds * time.Second):
			}
		}
	}()
}

func (w *ERxStatusWorker) Stop() {
	close(w.stopChan)
}

func (w *ERxStatusWorker) Do() error {
	msgs, err := w.erxQueue.QueueService.ReceiveMessage(w.erxQueue.QueueURL, nil, 1, msgVisibilityTimeout, longPollingTimePeriod)
	w.statCycles.Inc(1)
	if err != nil {
		w.statFailure.Inc(1)
		return err
	}

	if msgs == nil || len(msgs) == 0 {
		return nil
	}

	// keep track of failed events so as to determine
	// whether or not to delete a message from the queue
	startTime := time.Now()
	for _, msg := range msgs {
		statusCheckMessage := &common.PrescriptionStatusCheckMessage{}
		err := json.Unmarshal([]byte(msg.Body), statusCheckMessage)
		if err != nil {
			golog.Errorf("Unable to correctly parse json object for status check: %s", err.Error())
			w.statFailure.Inc(1)
			continue
		}

		patient, err := w.dataAPI.GetPatientFromID(statusCheckMessage.PatientID)
		if err != nil {
			golog.Errorf("Unable to get patient from database based on id: %s", err.Error())
			w.statFailure.Inc(1)
			continue
		}

		doctor, err := w.dataAPI.GetDoctorFromID(statusCheckMessage.DoctorID)
		if err != nil {
			golog.Errorf("Unable to get doctor from database based on id: %s", err.Error())
			w.statFailure.Inc(1)
			continue
		}

		var prescriptionStatuses []common.StatusEvent

		switch statusCheckMessage.EventCheckType {
		case common.RefillRxType:
			// check if there are any treatments for this patient that do not have a completed status
			prescriptionStatuses, err = w.dataAPI.GetApprovedOrDeniedRefillRequestsForPatient(patient.PatientID.Int64())
			if err != nil {
				golog.Errorf("Error getting prescription events for patient: %s", err.Error())
				w.statFailure.Inc(1)
				continue
			}
		case common.UnlinkedDNTFTreatmentType:
			prescriptionStatuses, err = w.dataAPI.GetErxStatusEventsForDNTFTreatmentBasedOnPatientID(patient.PatientID.Int64())
			if err != nil {
				golog.Errorf("Error getting prescriptiopn status events for dntf treatment for patient: %+v", err)
				w.statFailure.Inc(1)
				continue
			}
		case common.ERxType:
			// check if there are any treatments for this patient that do not have a completed status
			prescriptionStatuses, err = w.dataAPI.GetPrescriptionStatusEventsForPatient(patient.ERxPatientID.Int64())
			if err != nil {
				golog.Errorf("Error getting prescription events for patient: %+v", err)
				w.statFailure.Inc(1)
				continue
			}
		}

		// nothing to do if there are no prescriptions for this patient to keep track of
		if prescriptionStatuses == nil || len(prescriptionStatuses) == 0 {
			golog.Infof("No prescription statuses to keep track of for patient")
			err = w.erxQueue.QueueService.DeleteMessage(w.erxQueue.QueueURL, msg.ReceiptHandle)
			if err != nil {
				w.statFailure.Inc(1)
				golog.Errorf("Failed to delete message: %s", err.Error())
			}
			continue
		}

		// only hold on to the latest status event per treatment because that will help us
		// determine whether or not there are any treatments that do not have the end state
		// of the messages
		latestPendingStatusPerPrescription := make(map[int64]common.StatusEvent)
		for _, prescriptionStatus := range prescriptionStatuses {
			// the first occurence of every new event per prescription will be the latest because they are ordered by time
			if _, ok := latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionID]; !ok {
				// only keep track of tasks that have not reached the end state yet
				if prescriptionStatus.PrescriptionID != 0 {

					switch statusCheckMessage.EventCheckType {
					case common.RefillRxType:
						if prescriptionStatus.Status == api.RXRefillStatusApproved ||
							prescriptionStatus.Status == api.RXRefillStatusDenied {
							latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionID] = prescriptionStatus
						}
					case common.ERxType:
						if prescriptionStatus.Status == api.ERXStatusSending {
							latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionID] = prescriptionStatus
						}
					case common.UnlinkedDNTFTreatmentType:
						if prescriptionStatus.Status == api.ERXStatusSending {
							latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionID] = prescriptionStatus
						}
					}
				}
			}
		}

		if len(latestPendingStatusPerPrescription) == 0 {
			// nothing to do if there are no pending treatments to work with
			golog.Infof("There are no pending prescriptions for this patient")
			err = w.erxQueue.QueueService.DeleteMessage(w.erxQueue.QueueURL, msg.ReceiptHandle)
			if err != nil {
				w.statFailure.Inc(1)
				golog.Errorf("Failed to delete message: %s", err.Error())
			}
			continue
		}

		// or not to dequeue message from queue
		pendingTreatments := 0

		// go through each of the pending prescriptions to get log details to check if there have been any updates
		failed := 0
		for prescriptionID, prescriptionStatus := range latestPendingStatusPerPrescription {
			prescriptionLogs, err := w.erxAPI.GetPrescriptionStatus(doctor.DoseSpotClinicianID, prescriptionID)
			if err != nil {
				golog.Errorf("Unable to get log details for prescription id %d", prescriptionID)
				w.statFailure.Inc(1)
				failed++
				break
			}

			if len(prescriptionLogs) == 0 {
				pendingTreatments++
				continue
			}

			switch prescriptionLogs[0].PrescriptionStatus {
			case api.ERXStatusSending:
				// nothing to do
				pendingTreatments++
			case api.ERXStatusError:
				switch statusCheckMessage.EventCheckType {
				case common.RefillRxType:
					if err := w.dataAPI.AddRefillRequestStatusEvent(common.StatusEvent{
						Status:            api.RXRefillStatusError,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemID:            prescriptionStatus.ItemID,
						StatusDetails:     prescriptionLogs[0].AdditionalInfo,
					}); err != nil {
						w.statFailure.Inc(1)
						golog.Errorf("Unable to add status event for refill request: %+v", err)
						failed++
						break
					}
				case common.UnlinkedDNTFTreatmentType:
					if err := w.dataAPI.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
						Status:            api.ERXStatusError,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemID:            prescriptionStatus.ItemID,
						StatusDetails:     prescriptionLogs[0].AdditionalInfo,
					}); err != nil {
						w.statFailure.Inc(1)
						golog.Errorf("Unable to add status event for refill request: %+v", err)
						failed++
						break
					}
				case common.ERxType:
					treatment, err := w.dataAPI.GetTreatmentBasedOnPrescriptionID(prescriptionID)
					if err != nil {
						w.statFailure.Inc(1)
						golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
						failed++
						break
					}

					// get the error details for this medication
					if err := w.dataAPI.AddErxStatusEvent([]*common.Treatment{treatment}, common.StatusEvent{Status: api.ERXStatusError,
						StatusDetails:     prescriptionLogs[0].AdditionalInfo,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
					}); err != nil {
						w.statFailure.Inc(1)
						golog.Errorf("Unable to add error event for status: %s", err.Error())
						failed++
						break
					}
				}
				w.dispatcher.Publish(&RxTransmissionErrorEvent{
					DoctorID:  doctor.ID.Int64(),
					ItemID:    prescriptionStatus.ItemID,
					EventType: statusCheckMessage.EventCheckType,
					Patient:   patient,
				})
			case api.ERXStatusSent:
				switch statusCheckMessage.EventCheckType {
				case common.RefillRxType:
					if err := w.dataAPI.AddRefillRequestStatusEvent(common.StatusEvent{
						Status:            api.RXRefillStatusSent,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemID:            prescriptionStatus.ItemID,
					}); err != nil {
						w.statFailure.Inc(1)
						golog.Errorf("Unable to add status event for refill request: %+v", err)
						failed++
						break
					}
				case common.UnlinkedDNTFTreatmentType:
					if err := w.dataAPI.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
						Status:            api.ERXStatusSent,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemID:            prescriptionStatus.ItemID,
					}); err != nil {
						w.statFailure.Inc(1)
						golog.Errorf("Unable to add status event for refill request: %+v", err)
						failed++
						break
					}
				case common.ERxType:
					treatment, err := w.dataAPI.GetTreatmentBasedOnPrescriptionID(prescriptionID)
					if err != nil {
						w.statFailure.Inc(1)
						golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
						failed++
						break
					}

					// add an event
					err = w.dataAPI.AddErxStatusEvent([]*common.Treatment{treatment}, common.StatusEvent{Status: api.ERXStatusSent, ReportedTimestamp: prescriptionLogs[0].LogTimestamp})
					if err != nil {
						w.statFailure.Inc(1)
						golog.Errorf("Unable to add status event for this treatment: %s", err.Error())
						failed++
						break
					}
				}
			case api.ERXStatusDeleted:
				if statusCheckMessage.EventCheckType == common.RefillRxType {
					if err := w.dataAPI.AddRefillRequestStatusEvent(common.StatusEvent{
						Status:            api.RXRefillStatusDeleted,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemID:            prescriptionStatus.ItemID,
					}); err != nil {
						w.statFailure.Inc(1)
						golog.Errorf("Unable to add status event for refill request: %+v", err)
						failed++
						break
					}
				}
			default:
				pendingTreatments++
			}
		}

		if pendingTreatments == 0 && failed == 0 {
			// delete message from queue because there are no more pending treatments for this patient
			err = w.erxQueue.QueueService.DeleteMessage(
				w.erxQueue.QueueURL,
				msg.ReceiptHandle)
			if err != nil {
				w.statFailure.Inc(1)
				golog.Errorf("Failed to delete message: %s", err.Error())
			}
		}

	}
	responseTime := time.Since(startTime).Nanoseconds() / 1e3
	w.statProcessTime.Update(responseTime)
	return nil
}

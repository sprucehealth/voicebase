package app_worker

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"encoding/json"

	"github.com/samuel/go-metrics/metrics"

	"time"
)

const (
	waitTimeInMins        = 5
	msgVisibilityTimeout  = 30
	longPollingTimePeriod = 20
)

func StartWorkerToUpdatePrescriptionStatusForPatient(DataApi api.DataAPI, ERxApi erx.ERxAPI, ErxQueue *common.SQSQueue, statsRegistry metrics.Registry) {

	statProcessTime := metrics.NewBiasedHistogram()
	statCycles := metrics.NewCounter()
	statFailure := metrics.NewCounter()

	statsRegistry.Add("cycles/total", statCycles)
	statsRegistry.Add("cycles/processTime", statProcessTime)
	statsRegistry.Add("cycles/failed", statFailure)

	go func() {

		for {
			ConsumeMessageFromQueue(DataApi, ERxApi, ErxQueue, statProcessTime, statCycles, statFailure)
		}
	}()
}

func ConsumeMessageFromQueue(DataApi api.DataAPI, ERxApi erx.ERxAPI, ErxQueue *common.SQSQueue, statProcessTime metrics.Histogram, statCycles, statFailure metrics.Counter) {
	msgs, err := ErxQueue.QueueService.ReceiveMessage(ErxQueue.QueueUrl, nil, 1, msgVisibilityTimeout, longPollingTimePeriod)
	statCycles.Inc(1)
	if err != nil {
		golog.Errorf("Unable to receieve messages from queue. Sleeping and trying again in %d minutes", waitTimeInMins)
		time.Sleep(waitTimeInMins * time.Minute)
		statFailure.Inc(1)
		return
	}

	if msgs == nil || len(msgs) == 0 {
		time.Sleep(waitTimeInMins * time.Minute)
		statFailure.Inc(1)
		return
	}

	// keep track of failed events so as to determine
	// whether or not to delete a message from the queue
	startTime := time.Now()
	for _, msg := range msgs {
		statusCheckMessage := &common.PrescriptionStatusCheckMessage{}
		err := json.Unmarshal([]byte(msg.Body), statusCheckMessage)
		if err != nil {
			golog.Errorf("Unable to correctly parse json object for status check: %s", err.Error())
			statFailure.Inc(1)
			continue
		}

		patient, err := DataApi.GetPatientFromId(statusCheckMessage.PatientId)
		if err != nil {
			golog.Errorf("Unable to get patient from database based on id: %s", err.Error())
			statFailure.Inc(1)
			continue
		}

		doctor, err := DataApi.GetDoctorFromId(statusCheckMessage.DoctorId)
		if err != nil {
			golog.Errorf("Unable to get doctor from database based on id: %s", err.Error())
			statFailure.Inc(1)
			continue
		}

		var prescriptionStatuses []common.StatusEvent

		if statusCheckMessage.CheckRefillRequest {
			// check if there are any treatments for this patient that do not have a completed status
			prescriptionStatuses, err = DataApi.GetApprovedOrDeniedRefillRequestsForPatient(patient.PatientId.Int64())
			if err != nil {
				golog.Errorf("Error getting prescription events for patient: %s", err.Error())
				statFailure.Inc(1)
				continue
			}
		} else {
			// check if there are any treatments for this patient that do not have a completed status
			prescriptionStatuses, err = DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId.Int64())
			if err != nil {
				golog.Errorf("Error getting prescription events for patient: %s", err.Error())
				statFailure.Inc(1)
				continue
			}
		}

		// nothing to do if there are no prescriptions for this patient to keep track of
		if prescriptionStatuses == nil || len(prescriptionStatuses) == 0 {
			golog.Infof("No prescription statuses to keep track of for patient")
			err = ErxQueue.QueueService.DeleteMessage(ErxQueue.QueueUrl, msg.ReceiptHandle)
			if err != nil {
				statFailure.Inc(1)
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
			if _, ok := latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionId]; !ok {
				// only keep track of tasks that have not reached the end state yet
				if prescriptionStatus.PrescriptionId != 0 {

					if statusCheckMessage.CheckRefillRequest && (prescriptionStatus.Status == api.RX_REFILL_STATUS_APPROVED ||
						prescriptionStatus.Status == api.RX_REFILL_STATUS_DENIED) {
						latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionId] = prescriptionStatus
					}

					if !statusCheckMessage.CheckRefillRequest && prescriptionStatus.Status == api.ERX_STATUS_SENDING {
						latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionId] = prescriptionStatus
					}
				}
			}
		}

		if len(latestPendingStatusPerPrescription) == 0 {
			// nothing to do if there are no pending treatments to work with
			golog.Infof("There are no pending prescriptions for this patient")
			err = ErxQueue.QueueService.DeleteMessage(ErxQueue.QueueUrl, msg.ReceiptHandle)
			if err != nil {
				statFailure.Inc(1)
				golog.Errorf("Failed to delete message: %s", err.Error())
			}
			continue
		}

		// or not to dequeue message from queue
		pendingTreatments := 0

		// go through each of the pending prescriptions to get log details to check if there have been any updates
		failed := 0
		for prescriptionId, prescriptionStatus := range latestPendingStatusPerPrescription {
			prescriptionLogs, err := ERxApi.GetPrescriptionStatus(doctor.DoseSpotClinicianId, prescriptionId)
			if err != nil {
				golog.Errorf("Unable to get log details for prescription id %d", prescriptionId)
				statFailure.Inc(1)
				failed++
				break
			}

			if len(prescriptionLogs) == 0 {
				pendingTreatments++
				continue
			}

			switch prescriptionLogs[0].PrescriptionStatus {
			case api.ERX_STATUS_SENDING:
				// nothing to do
				pendingTreatments++
			case api.ERX_STATUS_ERROR:
				if statusCheckMessage.CheckRefillRequest {
					if err := DataApi.AddRefillRequestStatusEvent(common.StatusEvent{
						Status:            api.RX_REFILL_STATUS_ERROR,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemId:            prescriptionStatus.ItemId,
						StatusDetails:     prescriptionLogs[0].AdditionalInfo,
					}); err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to add status event for refill request: %+v", err)
						failed++
						break
					}

					if err := DataApi.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
						DoctorId:  doctor.DoctorId.Int64(),
						ItemId:    prescriptionStatus.ItemId,
						Status:    api.STATUS_PENDING,
						EventType: api.EVENT_TYPE_REFILL_TRANSMISSION_ERROR,
					}); err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to insert refill transmission error into doctor queue: %+v", err)
						failed++
						break
					}
				} else {
					treatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(prescriptionId)
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
						failed++
						break
					}

					// get the error details for this medication
					if err := DataApi.AddErxStatusEvent([]*common.Treatment{treatment}, common.StatusEvent{Status: api.ERX_STATUS_ERROR,
						StatusDetails:     prescriptionLogs[0].AdditionalInfo,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
					}); err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to add error event for status: %s", err.Error())
						failed++
						break
					}
					// insert an item into the doctor's queue to notify the doctor of this error
					if err := DataApi.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
						DoctorId:  doctor.DoctorId.Int64(),
						ItemId:    treatment.Id.Int64(),
						Status:    api.STATUS_PENDING,
						EventType: api.EVENT_TYPE_TRANSMISSION_ERROR,
					}); err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to insert error into doctor queue: %s", err.Error())
						failed++
						break
					}
				}
			case api.ERX_STATUS_SENT:
				if statusCheckMessage.CheckRefillRequest {
					if err := DataApi.AddRefillRequestStatusEvent(common.StatusEvent{
						Status:            api.RX_REFILL_STATUS_SENT,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemId:            prescriptionStatus.ItemId,
					}); err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to add status event for refill request: %+v", err)
						failed++
						break
					}
				} else {
					treatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(prescriptionId)
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
						failed++
						break
					}

					// add an event
					err = DataApi.AddErxStatusEvent([]*common.Treatment{treatment}, common.StatusEvent{Status: api.ERX_STATUS_SENT, ReportedTimestamp: prescriptionLogs[0].LogTimeStamp})
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to add status event for this treatment: %s", err.Error())
						failed++
						break
					}
				}
			case api.ERX_STATUS_DELETED:
				if statusCheckMessage.CheckRefillRequest {
					if err := DataApi.AddRefillRequestStatusEvent(common.StatusEvent{
						Status:            api.RX_REFILL_STATUS_DELETED,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemId:            prescriptionStatus.ItemId,
					}); err != nil {
						statFailure.Inc(1)
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
			err = ErxQueue.QueueService.DeleteMessage(ErxQueue.QueueUrl, msg.ReceiptHandle)
			if err != nil {
				statFailure.Inc(1)
				golog.Errorf("Failed to delete message: %s", err.Error())
			}
		}

	}
	responseTime := time.Since(startTime).Nanoseconds() / 1e3
	statProcessTime.Update(responseTime)
}

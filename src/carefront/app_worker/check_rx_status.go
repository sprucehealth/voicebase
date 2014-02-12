package app_worker

import (
	"carefront/api"
	"carefront/apiservice"
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
		statusCheckMessage := &apiservice.PrescriptionStatusCheckMessage{}
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

		// check if there are any treatments for this patient that do not have a completed status
		prescriptionStatuses, err := DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId)
		if err != nil {
			golog.Errorf("Error getting prescription events for patient: %s", err.Error())
			statFailure.Inc(1)
			continue
		}

		// nothing to do if there are no prescriptions for this patient to keep track of
		if prescriptionStatuses == nil || len(prescriptionStatuses) == 0 {
			golog.Infof("No prescription statuses to keep track of for patient")
			continue
		}

		// only hold on to the latest status event per treatment because that will help us
		// determine whether or not there are any treatments that do not have the end state
		// of the messages
		latestPendingStatusPerPrescription := make(map[int64]*api.PrescriptionStatus)
		for _, prescriptionStatus := range prescriptionStatuses {
			// the first occurence of every new event per prescription will be the latest because they are ordered by time
			if latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionId] == nil {
				// only keep track of tasks that have not reached the end state yet
				if prescriptionStatus.PrescriptionId != 0 && prescriptionStatus.PrescriptionStatus == api.ERX_STATUS_SENDING {
					latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionId] = prescriptionStatus
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

		medications, err := ERxApi.GetMedicationList(patient.ERxPatientId)
		if err != nil {
			golog.Errorf("Unable to get medications from dosespot: %s", err.Error())
			statFailure.Inc(1)
			continue
		}

		if medications == nil || len(medications) == 0 {
			golog.Infof("No medications returned for this patient from dosespot")
			err = ErxQueue.QueueService.DeleteMessage(ErxQueue.QueueUrl, msg.ReceiptHandle)
			if err != nil {
				statFailure.Inc(1)
				golog.Errorf("Failed to delete message: %s", err.Error())
			}
			continue
		}

		// keep track of treatments that are still pending for patient so that we know whether
		// or not to dequeue message from queue
		pendingTreatments := 0

		// go through treatments to see if the status has been updated to anything beyond sending
		failed := 0
		for _, medication := range medications {
			if latestPendingStatusPerPrescription[medication.ErxMedicationId] != nil {
				switch medication.PrescriptionStatus {
				case api.ERX_STATUS_SENDING:
					// nothing to do
					pendingTreatments++
				case api.ERX_STATUS_ERROR:
					treatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(medication.ErxMedicationId)
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
						failed++
						break
					}

					// get the error details for this medication
					medicationsWithErrors, err := ERxApi.GetTransmissionErrorDetails()
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to get transmission error details: %s", err.Error())
						failed++
						break
					}

					errorDetailsFound := false
					for _, medicationWithError := range medicationsWithErrors {
						// because of the nature of how the dosespot api is designed, getMedicationList returns the prescriptionId as the medicationId
						// and the getTransmissionErroDetails returns the prescriptionId as PrescriptionId
						if medicationWithError.DoseSpotPrescriptionId == medication.ErxMedicationId {
							errorDetailsFound = true
							golog.Infof("error found. here are the details: %s", medicationWithError.ErrorDetails)
							err = DataApi.AddErxErrorEventWithMessage(treatment, medication.PrescriptionStatus, medicationWithError.ErrorDetails, medicationWithError.ErrorTimeStamp)
							if err != nil {
								statFailure.Inc(1)
								golog.Errorf("Unable to add error event for status: %s", err.Error())
								failed++
								break
							}
							break
						}
					}
					if !errorDetailsFound {
						golog.Infof("error not found.")

						err = DataApi.AddErxStatusEvent([]*common.Treatment{treatment}, medication.PrescriptionStatus)
						if err != nil {
							statFailure.Inc(1)
							golog.Errorf("Unable to add error event for status: %s", err.Error())
							failed++
							break
						}
					}

				case api.ERX_STATUS_SENT:
					treatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(medication.ErxMedicationId)
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
						failed++
						break
					}

					// add an event
					err = DataApi.AddErxStatusEvent([]*common.Treatment{treatment}, medication.PrescriptionStatus)
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to add status event for this treatment: %s", err.Error())
						failed++
						break
					}

				}
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

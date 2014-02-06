package app_worker

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"encoding/json"
	"time"
)

const (
	waitTimeInMins        = 5
	msgVisibilityTimeout  = 30
	longPollingTimePeriod = 20
)

func StartWorkerToUpdatePrescriptionStatusForPatient(DataApi api.DataAPI, ERxApi erx.ERxAPI, ErxQueue *common.SQSQueue) {

	go func() {

		for {
			ConsumeMessageFromQueue(DataApi, ERxApi, ErxQueue)
		}
	}()
}

func ConsumeMessageFromQueue(DataApi api.DataAPI, ERxApi erx.ERxAPI, ErxQueue *common.SQSQueue) {
	msgs, err := ErxQueue.QueueService.ReceiveMessage(ErxQueue.QueueUrl, nil, 1, msgVisibilityTimeout, longPollingTimePeriod)
	if err != nil {
		golog.Errorf("Unable to receieve messages from queue. Sleeping and trying again in %d minutes", waitTimeInMins)
		time.Sleep(waitTimeInMins * time.Minute)
		return
	}

	if msgs == nil || len(msgs) == 0 {
		time.Sleep(waitTimeInMins * time.Minute)
		return
	}

	// keep track of failed events so as to determine
	// whether or not to delete a message from the queue
	failed := 0
	for _, msg := range msgs {
		statusCheckMessage := &apiservice.PrescriptionStatusCheckMessage{}
		err := json.Unmarshal([]byte(msg.Body), statusCheckMessage)
		if err != nil {
			golog.Errorf("Unable to correctly parse json object for status check: %s", err.Error())
			failed++
			continue
		}

		patient, err := DataApi.GetPatientFromId(statusCheckMessage.PatientId)
		if err != nil {
			golog.Errorf("Unable to get patient from database based on id: %s", err.Error())
			failed++
			continue
		}

		// check if there are any treatments for this patient that do not have a completed status
		prescriptionStatuses, err := DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId)
		if err != nil {
			golog.Errorf("Error getting prescription events for patient: %s", err.Error())
			failed++
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
			continue
		}

		medications, err := ERxApi.GetMedicationList(patient.ERxPatientId)
		if err != nil {
			golog.Errorf("Unable to get medications from dosespot: %s", err.Error())
			failed++
			continue
		}

		if medications == nil || len(medications) == 0 {
			golog.Infof("No medications returned for this patient from dosespot")
			continue
		}

		// keep track of treatments that are still pending for patient so that we know whether
		// or not to dequeue message from queue
		pendingTreatments := 0

		// go through treatments to see if the status has been updated to anything beyond sending
		for _, medication := range medications {
			if latestPendingStatusPerPrescription[medication.ErxMedicationId] != nil {
				switch medication.PrescriptionStatus {
				case api.ERX_STATUS_SENDING:
					// nothing to do
					pendingTreatments++
				case api.ERX_STATUS_SENT, api.ERX_STATUS_ERROR:
					// add an event
					treatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(medication.ErxMedicationId)
					if err != nil {
						golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
						failed++
						continue
					}
					err = DataApi.AddErxStatusEvent([]*common.Treatment{treatment}, medication.PrescriptionStatus)
					if err != nil {
						golog.Errorf("Unable to add status event for this treatment: %s", err.Error())
						failed++
						continue
					}
				}
			}
		}

		if pendingTreatments == 0 && failed == 0 {
			// delete message from queue because there are no more pending treatments for this patient
			err = ErxQueue.QueueService.DeleteMessage(ErxQueue.QueueUrl, msg.ReceiptHandle)
			if err != nil {
				golog.Warningf("Failed to delete message: %s", err.Error())
			}
		}
	}
}

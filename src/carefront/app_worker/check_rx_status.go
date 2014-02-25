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

type prescriptionType int

const (
	type_treatment prescriptionType = iota
	type_refill_request
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

		doctor, err := DataApi.GetDoctorFromId(statusCheckMessage.DoctorId)
		if err != nil {
			golog.Errorf("Unable to get doctor from database based on id: %s", err.Error())
			statFailure.Inc(1)
			continue
		}

		// check if there are any treatments for this patient that do not have a completed status
		prescriptionStatuses, err := DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId.Int64())
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
		prescriptionsToTrack := make(map[int64]prescriptionType)
		for _, prescriptionStatus := range prescriptionStatuses {
			// the first occurence of every new event per prescription will be the latest because they are ordered by time
			if prescriptionsToTrack[prescriptionStatus.PrescriptionId] == nil {
				// only keep track of tasks that have not reached the end state yet
				if prescriptionStatus.PrescriptionId != 0 && prescriptionStatus.PrescriptionStatus == api.ERX_STATUS_SENDING {
					prescriptionsToTrack[prescriptionStatus.PrescriptionId] = type_treatment
				}
			}
		}

		if len(prescriptionsToTrack) == 0 {
			// nothing to do if there are no pending treatments to work with
			golog.Infof("There are no pending prescriptions for this patient")
			err = ErxQueue.QueueService.DeleteMessage(ErxQueue.QueueUrl, msg.ReceiptHandle)
			if err != nil {
				statFailure.Inc(1)
				golog.Errorf("Failed to delete message: %s", err.Error())
			}
			continue
		}

		// check if there are any refill requests for this patient that do not have a completed or deleted status
		refillRequestStatuses, err := DataApi.GetRefillRequestsForPatientInGivenStates(patient.PatientId.Int64(),
			[]string{api.RX_REFILL_STATUS_APPROVED, api.RX_REFILL_STATUS_DENIED})

		if err != nil {
			golog.Errorf("Error getting refill request statuses for patient: %s", err.Error)
			statFailure.Inc(1)
			continue
		}

		prescriptionIdToRefillRequestMapping := make(map[int64]int64)
		for _, refillRequestStatus := range refillRequestStatuses {
			prescriptionsToTrack[refillRequestStatus.PrescriptionId] = type_refill_request
			prescriptionIdToRefillRequestMapping[refillRequestStatus.PrescriptionId] = refillRequestStatus.ErxRefillRequestId
		}

		medications, err := ERxApi.GetMedicationList(doctor.DoseSpotClinicianId, patient.ERxPatientId.Int64())
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
			if prescriptionsToTrack[medication.ErxMedicationId.Int64()] != nil {
				pType := prescriptionsToTrack[medication.ErxMedicationId.Int64()]
				switch medication.PrescriptionStatus {

				case api.ERX_STATUS_SENDING, api.ERX_STATUS_REQUESTED:
					// nothing to do
					pendingTreatments++

				case api.ERX_STATUS_ERROR:
					// get the error details for this medication
					prescriptionLogs, err := ERxApi.GetPrescriptionStatus(doctor.DoseSpotClinicianId, medication.ErxMedicationId.Int64())
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to get transmission error details: %s", err.Error())
						failed++
						break
					}

					errorDetailsFound := false
					var errorDetails string
					var errorDetailsTimestamp time.Time
					for _, prescriptionLog := range prescriptionLogs {
						// because of the nature of how the dosespot api is designed, getMedicationList returns the prescriptionId as the medicationId
						// and the getTransmissionErroDetails returns the prescriptionId as PrescriptionId
						if medication.PrescriptionStatus == prescriptionLog.PrescriptionStatus {
							errorDetailsFound = true
							errorDetails = prescriptionLog.AdditionalInfo
							errorDetailsTimestamp = prescriptionLog.LogTimeStamp
							break
						}
					}

					switch pType {
					case type_treatment:
						treatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(medication.ErxMedicationId.Int64())
						if err != nil {
							statFailure.Inc(1)
							golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
							failed++
							break
						}
						if errorDetailsFound {
							if err := DataApi.AddErxErrorEventWithMessage(treatment, medication.PrescriptionStatus, errorDetails, errorDetailsTimestamp); err != nil {
								statFailure.Inc(1)
								golog.Errorf("Unable to add error event for status: %s", err.Error())
								failed++
								break
							}
						} else {
							if err := DataApi.AddErxStatusEvent([]*common.Treatment{treatment}, medication.PrescriptionStatus); err != nil {
								statFailure.Inc(1)
								golog.Errorf("Unable to add error event for status: %s", err.Error())
								failed++
								break
							}
						}

					case type_refill_request:
						if errorDetailsFound {
							if err := DataApi.AddRefillRequestStatusEventWithMessage(prescriptionIdToRefillRequestMapping[medication.ErxMedicationId.Int64()], medication.PrescriptionStatus,
								errorDetails, errorDetailsTimestamp); err != nil {
								statFailure.Inc(1)
								golog.Errorf("Unable to add error event for refill request: %s", err.Error())
								failed++
								break
							}
						} else {
							if err := DataApi.AddRefillRequestStatusEvent(prescriptionIdToRefillRequestMapping[medication.ErxMedicationId.Int64()],
								medication.PrescriptionStatus, time.Now()); err != nil {
								statFailure.Inc(1)
								golog.Errorf("Unable to add event for refil request: %s", err.Error())
								failed++
								break
							}
						}
					}

				case api.ERX_STATUS_SENT:
					switch pType {
					case type_treatment:
						treatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(medication.ErxMedicationId.Int64())
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
					case type_refill_request:

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

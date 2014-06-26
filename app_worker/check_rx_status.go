package app_worker

import (
	"encoding/json"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"

	"time"
)

const (
	waitTimeInSeconds     = 30
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
		golog.Errorf("Unable to receieve messages from queue. Sleeping and trying again in %d seconds", waitTimeInSeconds)
		time.Sleep(waitTimeInSeconds * time.Second)
		statFailure.Inc(1)
		return
	}

	if msgs == nil || len(msgs) == 0 {
		time.Sleep(waitTimeInSeconds * time.Second)
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

		switch statusCheckMessage.EventCheckType {
		case common.RefillRxType:
			// check if there are any treatments for this patient that do not have a completed status
			prescriptionStatuses, err = DataApi.GetApprovedOrDeniedRefillRequestsForPatient(patient.PatientId.Int64())
			if err != nil {
				golog.Errorf("Error getting prescription events for patient: %s", err.Error())
				statFailure.Inc(1)
				continue
			}
		case common.UnlinkedDNTFTreatmentType:
			prescriptionStatuses, err = DataApi.GetErxStatusEventsForDNTFTreatmentBasedOnPatientId(patient.PatientId.Int64())
			if err != nil {
				golog.Errorf("Error getting prescriptiopn status events for dntf treatment for patient: %+v", err)
				statFailure.Inc(1)
			}
		case common.ERxType:
			// check if there are any treatments for this patient that do not have a completed status
			prescriptionStatuses, err = DataApi.GetPrescriptionStatusEventsForPatient(patient.ERxPatientId.Int64())
			if err != nil {
				golog.Errorf("Error getting prescription events for patient: %+v", err)
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

					switch statusCheckMessage.EventCheckType {
					case common.RefillRxType:
						if prescriptionStatus.Status == api.RX_REFILL_STATUS_APPROVED ||
							prescriptionStatus.Status == api.RX_REFILL_STATUS_DENIED {
							latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionId] = prescriptionStatus
						}
					case common.ERxType:
						if prescriptionStatus.Status == api.ERX_STATUS_SENDING {
							latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionId] = prescriptionStatus
						}
					case common.UnlinkedDNTFTreatmentType:
						if prescriptionStatus.Status == api.ERX_STATUS_SENDING {
							latestPendingStatusPerPrescription[prescriptionStatus.PrescriptionId] = prescriptionStatus
						}
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
				switch statusCheckMessage.EventCheckType {
				case common.RefillRxType:
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
				case common.UnlinkedDNTFTreatmentType:
					if err := DataApi.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
						Status:            api.ERX_STATUS_ERROR,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemId:            prescriptionStatus.ItemId,
						StatusDetails:     prescriptionLogs[0].AdditionalInfo,
					}); err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to add status event for refill request: %+v", err)
						failed++
						break
					}
				case common.ERxType:
					treatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(prescriptionId)
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
						failed++
						break
					}

					// get the error details for this medication
					if err := DataApi.AddErxStatusEvent([]int64{treatment.Id.Int64()}, common.StatusEvent{Status: api.ERX_STATUS_ERROR,
						StatusDetails:     prescriptionLogs[0].AdditionalInfo,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
					}); err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to add error event for status: %s", err.Error())
						failed++
						break
					}
				}
				dispatch.Default.Publish(&RxTransmissionErrorEvent{
					DoctorId:  doctor.DoctorId.Int64(),
					ItemId:    prescriptionStatus.ItemId,
					EventType: statusCheckMessage.EventCheckType,
				})
			case api.ERX_STATUS_SENT:
				switch statusCheckMessage.EventCheckType {
				case common.RefillRxType:
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
				case common.UnlinkedDNTFTreatmentType:
					if err := DataApi.AddErxStatusEventForDNTFTreatment(common.StatusEvent{
						Status:            api.ERX_STATUS_SENT,
						ReportedTimestamp: prescriptionLogs[0].LogTimestamp,
						ItemId:            prescriptionStatus.ItemId,
					}); err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to add status event for refill request: %+v", err)
						failed++
						break
					}
				case common.ERxType:
					treatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(prescriptionId)
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to get treatment based on prescription id: %s", err.Error())
						failed++
						break
					}

					// add an event
					err = DataApi.AddErxStatusEvent([]int64{treatment.Id.Int64()}, common.StatusEvent{Status: api.ERX_STATUS_SENT, ReportedTimestamp: prescriptionLogs[0].LogTimestamp})
					if err != nil {
						statFailure.Inc(1)
						golog.Errorf("Unable to add status event for this treatment: %s", err.Error())
						failed++
						break
					}
				}
			case api.ERX_STATUS_DELETED:
				if statusCheckMessage.EventCheckType == common.RefillRxType {
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

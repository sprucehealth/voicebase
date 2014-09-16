package doctor_treatment_plan

import (
	"encoding/json"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
)

type erxRouteMessage struct {
	TreatmentPlanID int64
	PatientID       int64
	DoctorID        int64
	Message         string
}

type worker struct {
	dataAPI         api.DataAPI
	erxAPI          erx.ERxAPI
	routeERx        bool
	erxRoutingQueue *common.SQSQueue
	erxStatusQueue  *common.SQSQueue
	timePeriod      int64
}

const (
	defaultTimePeriodSeconds           = 20
	visibilityTimeout                  = 30
	batchSize                          = 1
	successful_erx_routing_pharmacy_id = 47731
)

func StartWorker(dataAPI api.DataAPI, routeERx bool, erxAPI erx.ERxAPI, erxRoutingQueue *common.SQSQueue, erxStatusQueue *common.SQSQueue, timePeriod int64) {
	if timePeriod == 0 {
		timePeriod = defaultTimePeriodSeconds
	}

	(&worker{
		dataAPI:         dataAPI,
		erxAPI:          erxAPI,
		routeERx:        routeERx,
		erxRoutingQueue: erxRoutingQueue,
		erxStatusQueue:  erxStatusQueue,
		timePeriod:      timePeriod,
	}).start()
}

func (w *worker) start() {
	go func() {
		for {
			msgsConsumed, err := w.consumeMessage()

			if err != nil {
				golog.Errorf(err.Error())
			}

			if !msgsConsumed {
				time.Sleep(time.Duration(w.timePeriod) * time.Second)
			}
		}
	}()
}

func (w *worker) consumeMessage() (bool, error) {

	msgs, err := w.erxRoutingQueue.QueueService.ReceiveMessage(w.erxRoutingQueue.QueueUrl, nil, batchSize, visibilityTimeout, defaultTimePeriodSeconds)
	if err != nil {
		return false, err
	}

	if len(msgs) == 0 {
		return false, nil
	}

	msgsConsumed := true
	for _, msg := range msgs {
		routeMessage := erxRouteMessage{}
		if err := json.Unmarshal([]byte(msg.Body), &routeMessage); err != nil {
			golog.Errorf(err.Error())
			msgsConsumed = false
		}

		if err := w.processMessage(&routeMessage); err != nil {
			golog.Errorf(err.Error())
			msgsConsumed = false
		} else {
			if err := w.erxRoutingQueue.QueueService.DeleteMessage(w.erxRoutingQueue.QueueUrl, msg.ReceiptHandle); err != nil {
				golog.Errorf(err.Error())
				msgsConsumed = false
			}
		}
	}

	return msgsConsumed, nil
}

func (w *worker) processMessage(msg *erxRouteMessage) error {

	treatmentPlan, err := w.dataAPI.GetAbridgedTreatmentPlan(msg.TreatmentPlanID, msg.DoctorID)
	if err != nil {
		return err
	}
	currentTPStatus := treatmentPlan.Status

	treatments, err := w.dataAPI.GetTreatmentsBasedOnTreatmentPlanId(msg.TreatmentPlanID)
	if err != nil {
		return err
	}

	doctor, err := w.dataAPI.GetDoctorFromId(msg.DoctorID)
	if err != nil {
		return err
	}

	patient, err := w.dataAPI.GetPatientFromId(msg.PatientID)
	if err != nil {
		return err
	}

	// activate the treatment plan and send the case message if we are not routing e-prescriptions
	// or there are no treatments in the TP
	if !w.routeERx || len(treatments) == 0 {
		if err := w.dataAPI.ActivateTreatmentPlan(treatmentPlan.Id.Int64(), doctor.DoctorId.Int64()); err != nil {
			return err
		}

		if err := w.sendCaseMessageAndPublishTPActivatedEvent(treatmentPlan, doctor, patient, msg.Message); err != nil {
			return err
		}

		return nil
	}

	// Route the prescriptions if the treatment plan is in the submitted state
	if currentTPStatus == common.TPStatusSubmitted {

		// its possible for the call to start prescribing medications to have succeeded
		// previously but the call to update the treamtent plan status to have failed, however,
		// given that prescriptions are not sent until we actually call the send prescriptions
		// API, its okay to make the call to start prescribing again
		if err := w.erxAPI.StartPrescribingPatient(doctor.DoseSpotClinicianId,
			patient, treatments, patient.Pharmacy.SourceId); err != nil {
			return err
		}

		if patient.ERxPatientId.Int64() == 0 {
			if err := w.dataAPI.UpdatePatientWithERxPatientId(patient.PatientId.Int64(), patient.ERxPatientId.Int64()); err != nil {
				return err
			}
		}

		// update the treatments to have the prescription ids and also track the pharmacy to which the prescriptions will be sent
		// at the same time, update the status of the treatment plan to indicate that we succesfullly
		// start prescribing prescriptions for this patient
		if err := w.dataAPI.StartRXRoutingForTreatmentsAndTreatmentPlan(treatments, patient.Pharmacy, treatmentPlan.Id.Int64(), doctor.DoctorId.Int64()); err != nil {
			return err
		}

		currentTPStatus = common.TPStatusRXStarted
	}

	if currentTPStatus == common.TPStatusRXStarted {
		// given that we make the call to send prescriptions as one API call,
		// lets check the status of one of the treatments to understand
		// whether or not the prescriptions have already been sent
		prescriptionLogs, err := w.erxAPI.GetPrescriptionStatus(doctor.DoseSpotClinicianId, treatments[0].ERx.PrescriptionId.Int64())
		if err != nil {
			return err
		}

		// only send the prescriptions to the pharmacy if the treatment is in the entered state
		if len(prescriptionLogs) == 1 && prescriptionLogs[0].PrescriptionStatus == api.ERX_STATUS_ENTERED {
			if err := w.sendPrescriptionsToPharmacy(treatments, patient, doctor); err != nil {
				return err
			}
		}

		if err := w.dataAPI.ActivateTreatmentPlan(treatmentPlan.Id.Int64(), doctor.DoctorId.Int64()); err != nil {
			return err
		}
		currentTPStatus = common.TPStatusActive
	}

	if err := w.sendCaseMessageAndPublishTPActivatedEvent(treatmentPlan, doctor, patient, msg.Message); err != nil {
		return err
	}

	return nil
}

func (w *worker) sendCaseMessageAndPublishTPActivatedEvent(treatmentPlan *common.DoctorTreatmentPlan,
	doctor *common.Doctor, patient *common.Patient, message string) error {
	// only send a case message if one has not already been sent for this particular
	// treatment plan for this particular case
	caseMessage, err := w.dataAPI.CaseMessageForAttachment(common.AttachmentTypeTreatmentPlan,
		treatmentPlan.Id.Int64(), doctor.PersonId, treatmentPlan.PatientCaseId.Int64())
	if err != api.NoRowsError && err != nil {
		return err
	} else if err == api.NoRowsError {
		caseMessage = &common.CaseMessage{
			CaseID:   treatmentPlan.PatientCaseId.Int64(),
			PersonID: doctor.PersonId,
			Body:     message,
			Attachments: []*common.CaseMessageAttachment{
				&common.CaseMessageAttachment{
					ItemType: common.AttachmentTypeTreatmentPlan,
					ItemID:   treatmentPlan.Id.Int64(),
				},
			},
		}
		if _, err := w.dataAPI.CreateCaseMessage(caseMessage); err != nil {
			return err
		}
	}

	patientVisitID, err := w.dataAPI.GetPatientVisitIdFromTreatmentPlanId(treatmentPlan.Id.Int64())
	if err != nil {
		return err
	}

	// Publish event that treamtent plan was created
	dispatch.Default.Publish(&TreatmentPlanActivatedEvent{
		PatientId:     treatmentPlan.PatientId,
		DoctorId:      doctor.DoctorId.Int64(),
		VisitId:       patientVisitID,
		TreatmentPlan: treatmentPlan,
		Patient:       patient,
		Message:       caseMessage,
	})

	return nil
}

func (w *worker) sendPrescriptionsToPharmacy(treatments []*common.Treatment, patient *common.Patient, doctor *common.Doctor) error {

	// Now, request the medications to be sent to the patient's preferred pharmacy
	unSuccessfulTreatments, err := w.erxAPI.SendMultiplePrescriptions(doctor.DoseSpotClinicianId, patient, treatments)
	if err != nil {
		return err
	}

	// gather treatmentIds for treatments that were successfully routed to pharmacy
	successfulTreatments := make([]*common.Treatment, 0, len(treatments))
	for _, treatment := range treatments {
		treatmentFound := false
		for _, unSuccessfulTreatment := range unSuccessfulTreatments {
			if unSuccessfulTreatment.Id.Int64() == treatment.Id.Int64() {
				treatmentFound = true
				break
			}
		}
		if !treatmentFound {
			successfulTreatments = append(successfulTreatments, treatment)
		}
	}

	if err := w.dataAPI.AddErxStatusEvent(successfulTreatments, common.StatusEvent{Status: api.ERX_STATUS_SENDING}); err != nil {
		return err
	}

	if err := w.dataAPI.AddErxStatusEvent(unSuccessfulTreatments, common.StatusEvent{Status: api.ERX_STATUS_SEND_ERROR}); err != nil {
		return err
	}

	//  Queue up notification to patient
	if err := apiservice.QueueUpJob(w.erxStatusQueue, &common.PrescriptionStatusCheckMessage{
		PatientId:      patient.PatientId.Int64(),
		DoctorId:       doctor.DoctorId.Int64(),
		EventCheckType: common.ERxType,
	}); err != nil {
		golog.Errorf("Unable to enqueue job to check status of erx. Not going to error out on this for the user because there is nothing the user can do about this: %+v", err)
	}
	return nil
}

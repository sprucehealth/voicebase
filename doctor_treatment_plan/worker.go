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
	erxRoutingQueue *common.SQSQueue
	erxStatusQueue  *common.SQSQueue
	timePeriod      int64
}

const (
	defaultTimePeriodSeconds = 20
	visibilityTimeout        = 30
	batchSize                = 1
)

func StartWorker(dataAPI api.DataAPI, erxAPI erx.ERxAPI, erxRoutingQueue *common.SQSQueue, erxStatusQueue *common.SQSQueue, timePeriod int64) {
	if timePeriod == 0 {
		timePeriod = defaultTimePeriodSeconds
	}

	(&worker{
		dataAPI:         dataAPI,
		erxAPI:          erxAPI,
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
			msgsConsumed = false
		}

		if err := w.processMessage(&routeMessage); err != nil {
			golog.Errorf(err.Error())
			msgsConsumed = false
		} else {
			if err := w.erxRoutingQueue.QueueService.DeleteMessage(w.erxRoutingQueue.QueueUrl, msg.ReceiptHandle); err != nil {
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

	treatments, err := w.dataAPI.GetTreatmentsBasedOnTreatmentPlanId(msg.TreatmentPlanID)
	if err != nil {
		return err
	}

	patient, err := w.dataAPI.GetPatientFromId(msg.PatientID)
	if err != nil {
		return err
	}

	doctor, err := w.dataAPI.GetDoctorFromId(msg.DoctorID)
	if err != nil {
		return err
	}

	// TODO only call start prescribing patient if the treatments don't already have an erx id

	// Send the patient information and new medications to be prescribed, to dosespot
	if err := w.erxAPI.StartPrescribingPatient(doctor.DoseSpotClinicianId, patient, treatments, patient.Pharmacy.SourceId); err != nil {
		return err
	}

	if patient.ERxPatientId.Int64() == 0 {
		// Save erx patient id to database once we get it back from dosespot
		if err := w.dataAPI.UpdatePatientWithERxPatientId(patient.PatientId.Int64(), patient.ERxPatientId.Int64()); err != nil {
			return err
		}
	}

	// Save prescription ids for drugs to database
	if err := w.dataAPI.UpdateTreatmentWithPharmacyAndErxId(treatments, patient.Pharmacy, doctor.DoctorId.Int64()); err != nil {
		return err
	}

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

	if err := w.dataAPI.ActivateTreatmentPlan(msg.TreatmentPlanID, doctor.DoctorId.Int64()); err != nil {
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

	caseMessage := &common.CaseMessage{
		CaseID:   treatmentPlan.PatientCaseId.Int64(),
		PersonID: doctor.PersonId,
		Body:     msg.Message,
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

	patientVisitID, err := w.dataAPI.GetPatientVisitIdFromTreatmentPlanId(msg.TreatmentPlanID)
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

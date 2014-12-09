package doctor_treatment_plan

import (
	"encoding/json"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/errors"
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
	dispatcher      *dispatch.Dispatcher
	erxRoutingQueue *common.SQSQueue
	erxStatusQueue  *common.SQSQueue
	erxRouteFail    *metrics.Counter
	erxRouteSuccess *metrics.Counter
	timePeriod      int64
}

const (
	defaultTimePeriodSeconds           = 20
	visibilityTimeout                  = 30
	batchSize                          = 1
	successful_erx_routing_pharmacy_id = 47731
)

func StartWorker(dataAPI api.DataAPI, erxAPI erx.ERxAPI, dispatcher *dispatch.Dispatcher, erxRoutingQueue *common.SQSQueue, erxStatusQueue *common.SQSQueue, timePeriod int64, metricsRegistry metrics.Registry) {
	if timePeriod == 0 {
		timePeriod = defaultTimePeriodSeconds
	}

	erxRouteFail := metrics.NewCounter()
	erxRouteSuccess := metrics.NewCounter()
	metricsRegistry.Add("route/failure", erxRouteFail)
	metricsRegistry.Add("route/success", erxRouteSuccess)

	w := &worker{
		dataAPI:         dataAPI,
		erxAPI:          erxAPI,
		dispatcher:      dispatcher,
		erxRoutingQueue: erxRoutingQueue,
		erxStatusQueue:  erxStatusQueue,
		timePeriod:      timePeriod,
		erxRouteFail:    erxRouteFail,
		erxRouteSuccess: erxRouteSuccess,
	}

	if environment.IsTest() {
		w.consumeMessage()
	} else {
		w.start()
	}
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
	msgs, err := w.erxRoutingQueue.QueueService.ReceiveMessage(w.erxRoutingQueue.QueueURL, nil, batchSize, visibilityTimeout, defaultTimePeriodSeconds)
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
			if err := w.erxRoutingQueue.QueueService.DeleteMessage(w.erxRoutingQueue.QueueURL, msg.ReceiptHandle); err != nil {
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
		return errors.Trace(err)
	}
	currentTPStatus := treatmentPlan.Status

	treatments, err := w.dataAPI.GetTreatmentsBasedOnTreatmentPlanID(msg.TreatmentPlanID)
	if err != nil {
		return errors.Trace(err)
	}

	doctor, err := w.dataAPI.GetDoctorFromID(msg.DoctorID)
	if err != nil {
		return errors.Trace(err)
	}

	patient, err := w.dataAPI.GetPatientFromID(msg.PatientID)
	if err != nil {
		return errors.Trace(err)
	}

	// activate the treatment plan and send the case message if we are not routing e-prescriptions
	// or there are no treatments in the TP
	if len(treatments) == 0 {
		if err := w.dataAPI.ActivateTreatmentPlan(treatmentPlan.ID.Int64(), doctor.DoctorID.Int64()); err != nil {
			return errors.Trace(err)
		}

		if err := sendCaseMessageAndPublishTPActivatedEvent(w.dataAPI, w.dispatcher, treatmentPlan, doctor, msg.Message); err != nil {
			return errors.Trace(err)
		}

		return nil
	}

	// Route the prescriptions if the treatment plan is in the submitted state
	if currentTPStatus == common.TPStatusSubmitted {

		// its possible for the call to start prescribing medications to have succeeded
		// previously but the call to update the treamtent plan status to have failed, however,
		// given that prescriptions are not sent until we actually call the send prescriptions
		// API, its okay to make the call to start prescribing again
		if err := w.erxAPI.StartPrescribingPatient(doctor.DoseSpotClinicianID,
			patient, treatments, patient.Pharmacy.SourceID); err != nil {
			w.erxRouteFail.Inc(1)
			return errors.Trace(err)
		}

		if err := w.dataAPI.UpdatePatientWithERxPatientID(patient.PatientID.Int64(), patient.ERxPatientID.Int64()); err != nil {
			return errors.Trace(err)
		}

		// update the treatments to have the prescription ids and also track the pharmacy to which the prescriptions will be sent
		// at the same time, update the status of the treatment plan to indicate that we succesfullly
		// start prescribing prescriptions for this patient
		if err := w.dataAPI.StartRXRoutingForTreatmentsAndTreatmentPlan(treatments, patient.Pharmacy, treatmentPlan.ID.Int64(), doctor.DoctorID.Int64()); err != nil {
			return errors.Trace(err)
		}

		currentTPStatus = common.TPStatusRXStarted
	}

	if currentTPStatus == common.TPStatusRXStarted {

		if err := w.sendPrescriptionsToPharmacy(treatments, patient, doctor); err != nil {
			return errors.Trace(err)
		}

		if err := w.dataAPI.ActivateTreatmentPlan(treatmentPlan.ID.Int64(), doctor.DoctorID.Int64()); err != nil {
			return errors.Trace(err)
		}
		currentTPStatus = common.TPStatusActive
	}

	if err := sendCaseMessageAndPublishTPActivatedEvent(w.dataAPI, w.dispatcher, treatmentPlan, doctor, msg.Message); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (w *worker) sendPrescriptionsToPharmacy(treatments []*common.Treatment, patient *common.Patient, doctor *common.Doctor) error {
	prescriptionsToSend, err := w.determinePrescriptionsToSendToPharmacy(treatments, doctor)
	if err != nil {
		return errors.Trace(err)
	} else if len(prescriptionsToSend) == 0 {
		return nil
	}

	// Now, request the medications to be sent to the patient's preferred pharmacy
	unSuccessfulTreatments, err := w.erxAPI.SendMultiplePrescriptions(doctor.DoseSpotClinicianID, patient, prescriptionsToSend)
	if err != nil {
		w.erxRouteFail.Inc(1)
		return errors.Trace(err)
	} else if len(unSuccessfulTreatments) > 0 {
		w.erxRouteFail.Inc(1)
	}

	// gather treatmentIds for treatments that were successfully routed to pharmacy
	successfulTreatments := make([]*common.Treatment, 0, len(treatments))
	for _, treatment := range treatments {
		treatmentFound := false
		for _, unSuccessfulTreatment := range unSuccessfulTreatments {
			if unSuccessfulTreatment.ID.Int64() == treatment.ID.Int64() {
				treatmentFound = true
				break
			}
		}
		if !treatmentFound {
			successfulTreatments = append(successfulTreatments, treatment)
		}
	}

	if err := w.dataAPI.AddErxStatusEvent(successfulTreatments, common.StatusEvent{Status: api.ERX_STATUS_SENDING}); err != nil {
		return errors.Trace(err)
	}

	if err := w.dataAPI.AddErxStatusEvent(unSuccessfulTreatments, common.StatusEvent{Status: api.ERX_STATUS_SEND_ERROR}); err != nil {
		return errors.Trace(err)
	}

	//  Queue up notification to patient
	if err := apiservice.QueueUpJob(w.erxStatusQueue, &common.PrescriptionStatusCheckMessage{
		PatientID:      patient.PatientID.Int64(),
		DoctorID:       doctor.DoctorID.Int64(),
		EventCheckType: common.ERxType,
	}); err != nil {
		golog.Errorf("Unable to enqueue job to check status of erx. Not going to error out on this for the user because there is nothing the user can do about this: %+v", err)
	}
	w.erxRouteSuccess.Inc(1)
	return nil
}

func (w *worker) determinePrescriptionsToSendToPharmacy(treatments []*common.Treatment, doctor *common.Doctor) ([]*common.Treatment, error) {
	var treatmentsToSend []*common.Treatment
	for _, tItem := range treatments {
		prescriptionLogs, err := w.erxAPI.GetPrescriptionStatus(doctor.DoseSpotClinicianID, tItem.ERx.PrescriptionID.Int64())
		if err != nil {
			return nil, errors.Trace(err)
		}

		// only send the prescriptions to the pharmacy if the treatment is in the entered state
		if len(prescriptionLogs) == 1 && prescriptionLogs[0].PrescriptionStatus == api.ERX_STATUS_ENTERED {
			treatmentsToSend = append(treatmentsToSend, tItem)
		}
	}
	return treatmentsToSend, nil
}

package notify

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/common/config"
	"carefront/libs/aws/sns"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"carefront/messages"
	"errors"

	"github.com/samuel/go-metrics/metrics"
	"github.com/subosito/twilio"
)

var manager notificationManager

type notificationManager struct {
	dataApi             api.DataAPI
	snsClient           *sns.SNS
	twilioClient        *twilio.Client
	fromNumber          string
	notificationConfigs map[string]*config.NotificationConfig
	statSMSSent         metrics.Counter
	statSMSFailed       metrics.Counter
	statPushSent        metrics.Counter
	statPushFailed      metrics.Counter
}

func InitManager(dataApi api.DataAPI, snsClient *sns.SNS, twilioClient *twilio.Client, fromNumber string, notificationConfigs map[string]*config.NotificationConfig, statsRegistry metrics.Registry) {

	manager = notificationManager{
		dataApi:             dataApi,
		snsClient:           snsClient,
		twilioClient:        twilioClient,
		fromNumber:          fromNumber,
		notificationConfigs: notificationConfigs,
		statSMSSent:         metrics.NewCounter(),
		statSMSFailed:       metrics.NewCounter(),
		statPushSent:        metrics.NewCounter(),
		statPushFailed:      metrics.NewCounter(),
	}

	statsRegistry.Scope("twilio").Add("sms/sent", manager.statSMSSent)
	statsRegistry.Scope("twilio").Add("sms/failed", manager.statSMSFailed)
	statsRegistry.Scope("sns").Add("push/sent", manager.statPushSent)
	statsRegistry.Scope("sns").Add("push/failed", manager.statPushFailed)

	dispatch.Default.Subscribe(func(ev *apiservice.VisitSubmittedEvent) error {
		doctor, err := dataApi.GetDoctorFromId(ev.DoctorId)
		if err != nil {
			return err
		}

		if err := manager.notifyDoctor(doctor, ev); err != nil {
			return err
		}

		return nil
	})

	// Notify the patient when the doctor has reviewed the visit and submitted a treatment plan
	dispatch.Default.Subscribe(func(ev *apiservice.VisitReviewSubmittedEvent) error {
		patient := ev.Patient
		if patient == nil {
			var err error
			patient, err = manager.dataApi.GetPatientFromId(ev.PatientId)
			if err != nil {
				golog.Errorf("notify: failed to get patient %d: %s", ev.PatientId, err.Error())
				return err
			}
		}

		if err := manager.notifyPatient(patient, ev); err != nil {
			return err
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *messages.ConversationStartedEvent) error {
		people, err := manager.dataApi.GetPeople([]int64{ev.FromId, ev.ToId})
		if err != nil {
			return err
		} else if len(people) != 2 {
			return errors.New("failed to find person for conversation")
		}

		from := people[ev.FromId]
		to := people[ev.ToId]

		if to.RoleType == api.PATIENT_ROLE && from.RoleType == api.DOCTOR_ROLE {
			if err := manager.notifyPatient(to.Patient, ev); err != nil {
				return err
			}
		} else if to.RoleType == api.DOCTOR_ROLE && from.RoleType == api.DOCTOR_ROLE {
			if err := manager.notifyDoctor(to.Doctor, ev); err != nil {
				return err
			}
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *messages.ConversationReplyEvent) error {
		con, err := manager.dataApi.GetConversation(ev.ConversationId)
		if err != nil {
			return err
		}
		from := con.Participants[ev.FromId]
		if from == nil {
			return errors.New("failed to find person conversation is from")
		}

		var doctorPerson *common.Person
		var patientPerson *common.Person
		for _, p := range con.Participants {
			switch p.RoleType {
			case api.PATIENT_ROLE:
				patientPerson = p
			case api.DOCTOR_ROLE:
				doctorPerson = p
			}
		}

		if from.RoleType != api.PATIENT_ROLE && patientPerson != nil {
			// Notify patient
			if err := manager.notifyPatient(patientPerson.Patient, ev); err != nil {
				return err
			}
		} else if from.RoleType != api.DOCTOR_ROLE && doctorPerson != nil {
			// Notify doctor
			if err := manager.notifyDoctor(doctorPerson.Doctor, ev); err != nil {
				return err
			}
		}

		return nil
	})
}

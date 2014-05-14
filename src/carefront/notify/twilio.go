package notify

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"carefront/messages"
	"errors"
	"fmt"

	"github.com/samuel/go-metrics/metrics"
	"github.com/subosito/twilio"
)

const (
	doctorNewVisitNotification     = "SPRUCE: You have a new patient visit waiting."
	patientVisitUpdateNotification = "SPRUCE: There is an update to your case."
	newMessageNotification         = "SPRUCE: You have a new message."
)

func phoneNumberForPatient(patient *common.Patient) string {
	if patient == nil {
		return ""
	}
	for _, phoneNumber := range patient.PhoneNumbers {
		if phoneNumber.PhoneType == api.PHONE_CELL {
			return patient.PhoneNumbers[0].Phone
		}
	}
	return ""
}

func InitTwilio(dataAPI api.DataAPI, twilioCli *twilio.Client, fromNumber, iosDeeplinkScheme string, statsRegistry metrics.Registry) {
	if twilioCli == nil {
		return
	}

	statSMSSent := metrics.NewCounter()
	statSMSFailed := metrics.NewCounter()
	statsRegistry.Add("sms/sent", statSMSSent)
	statsRegistry.Add("sms/failed", statSMSFailed)

	// Notify the doctor when a patient submits a new visit
	dispatch.Default.Subscribe(func(ev *apiservice.VisitSubmittedEvent) error {
		if doc, err := dataAPI.GetDoctorFromId(ev.DoctorId); err != nil {
			return fmt.Errorf("notify: failed to get doctor %d: %s", ev.DoctorId, err.Error())
		} else if doc.CellPhone != "" {
			_, _, err = twilioCli.Messages.SendSMS(fromNumber, doc.CellPhone, doctorNewVisitNotification)
			if err == nil {
				statSMSSent.Inc(1)
			} else if err != nil {
				statSMSFailed.Inc(1)
				return fmt.Errorf("notify: error sending SMS to %s: %s", doc.CellPhone, err.Error())
			}
		}
		return nil
	})

	// Notify the patient when the doctor has reviewed the visit and submitted a treatment plan
	dispatch.Default.Subscribe(func(ev *apiservice.VisitReviewSubmittedEvent) error {
		patient := ev.Patient
		if patient == nil {
			var err error
			patient, err = dataAPI.GetPatientFromId(ev.PatientId)
			if err != nil {
				return fmt.Errorf("notify: failed to get patient %d: %s", ev.PatientId, err.Error())
			}
		}
		if len(patient.PhoneNumbers) > 0 {
			if toNumber := phoneNumberForPatient(patient); toNumber != "" {
				_, _, err := twilioCli.Messages.SendSMS(fromNumber, toNumber, patientVisitUpdateNotification)
				if err == nil {
					statSMSSent.Inc(1)
				} else if err != nil {
					statSMSFailed.Inc(1)
					return fmt.Errorf("notify: error sending SMS: %s", err.Error())
				}
			}
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *messages.ConversationStartedEvent) error {
		people, err := dataAPI.GetPeople([]int64{ev.FromId, ev.ToId})
		if err != nil {
			return err
		}
		if len(people) != 2 {
			return errors.New("failed to find person for conversation")
		}
		from := people[ev.FromId]
		to := people[ev.ToId]

		if to.RoleType == api.PATIENT_ROLE && from.RoleType == api.DOCTOR_ROLE {
			// Notify patient
			if toNumber := phoneNumberForPatient(to.Patient); toNumber != "" {
				_, _, err := twilioCli.Messages.SendSMS(fromNumber, toNumber, newMessageNotification)
				if err == nil {
					statSMSSent.Inc(1)
				} else if err != nil {
					statSMSFailed.Inc(1)
					golog.Errorf("notify: error sending SMS: %s", err.Error())
				}
			}
		} else if to.RoleType == api.DOCTOR_ROLE && from.RoleType == api.DOCTOR_ROLE {
			// Notify doctor
			if toNumber := to.Doctor.CellPhone; toNumber != "" {
				_, _, err := twilioCli.Messages.SendSMS(fromNumber, toNumber, newMessageNotification)
				if err == nil {
					statSMSSent.Inc(1)
				} else if err != nil {
					statSMSFailed.Inc(1)
					golog.Errorf("notify: error sending SMS: %s", err.Error())
				}
			}
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *messages.ConversationReplyEvent) error {
		con, err := dataAPI.GetConversation(ev.ConversationId)
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
			if toNumber := phoneNumberForPatient(patientPerson.Patient); toNumber != "" {
				_, _, err := twilioCli.Messages.SendSMS(fromNumber, toNumber, newMessageNotification)
				if err == nil {
					statSMSSent.Inc(1)
				} else if err != nil {
					statSMSFailed.Inc(1)
					golog.Errorf("notify: error sending SMS: %s", err.Error())
				}
			}
		} else if from.RoleType != api.DOCTOR_ROLE && doctorPerson != nil {
			// Notify doctor
			if toNumber := doctorPerson.Doctor.CellPhone; toNumber != "" {
				_, _, err := twilioCli.Messages.SendSMS(fromNumber, toNumber, newMessageNotification)
				if err == nil {
					statSMSSent.Inc(1)
				} else if err != nil {
					statSMSFailed.Inc(1)
					golog.Errorf("notify: error sending SMS: %s", err.Error())
				}
			}
		}

		return nil
	})
}

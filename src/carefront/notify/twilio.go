package notify

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/libs/dispatch"
	"fmt"

	"github.com/subosito/twilio"
)

const (
	doctorNewVisitNotification     = "SPRUCE: You have a new patient visit waiting."
	patientVisitUpdateNotification = "There is an update to your case. Tap %s://visit to view."
)

func InitTwilio(dataApi api.DataAPI, twilioCli *twilio.Client, fromNumber, iosDeeplinkScheme string) {
	if twilioCli == nil {
		return
	}

	// Notify the doctor when a patient submits a new visit
	dispatch.Default.Subscribe(func(ev *apiservice.VisitSubmittedEvent) error {
		if doc, err := dataApi.GetDoctorFromId(ev.DoctorId); err != nil {
			return fmt.Errorf("notify: failed to get doctor %d: %s", ev.DoctorId, err.Error())
		} else if doc.CellPhone != "" {
			_, _, err = twilioCli.Messages.SendSMS(fromNumber, doc.CellPhone, doctorNewVisitNotification)
			if err != nil {
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
			patient, err = dataApi.GetPatientFromId(ev.PatientId)
			if err != nil {
				return fmt.Errorf("notify: failed to get patient %d: %s", ev.PatientId, err.Error())
			}
		}
		if len(patient.PhoneNumbers) > 0 {
			for _, phoneNumber := range patient.PhoneNumbers {
				if phoneNumber.PhoneType == api.PHONE_CELL {
					toNumber := patient.PhoneNumbers[0].Phone
					_, _, err := twilioCli.Messages.SendSMS(fromNumber, toNumber, fmt.Sprintf(patientVisitUpdateNotification, iosDeeplinkScheme))
					if err != nil {
						return fmt.Errorf("notify: error sending SMS to %s: %s", toNumber, err.Error())
					}
				}
			}
		}
		return nil
	})
}

package patient_case

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/patient_visit"
)

func InitListeners(dataAPI api.DataAPI, notificationManager *notify.NotificationManager) {
	dispatch.Default.Subscribe(func(ev *messages.PostEvent) error {

		// delete any pending visit submitted notifications for case
		if err := dataAPI.DeleteCaseNotification(CNVisitSubmitted, ev.Case.Id.Int64()); err != nil {
			golog.Errorf("Unable to delete case notifications: %s", err)
			return err
		}

		// insert notification into patient case if the doctor
		// sent the patient a message
		if ev.Person.RoleType == api.DOCTOR_ROLE {
			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseId:    ev.Case.Id.Int64(),
				NotificationType: CNMessage,
				UID:              fmt.Sprintf("%s:%d", CNMessage, ev.Message.ID),
				Data: &messageNotification{
					MessageId: ev.Message.ID,
					DoctorId:  ev.Person.Doctor.DoctorId.Int64(),
					CaseId:    ev.Message.CaseID,
				},
			}); err != nil {
				golog.Errorf("Unable to insert notification item for case: %s", err)
				return err
			}

			patient, err := dataAPI.GetPatientFromId(ev.Case.PatientId.Int64())
			if err != nil {
				golog.Errorf("Unable to get patient from id: %s", err)
				return err
			}

			// notify the patient of the message
			if err := notificationManager.NotifyPatient(patient, ev); err != nil {
				golog.Errorf("Unable to notify patient: %s", err)
				return err
			}
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {
		// delete any pending visit submitted notifications for case
		if err := dataAPI.DeleteCaseNotification(CNVisitSubmitted, ev.TreatmentPlan.PatientCaseId.Int64()); err != nil {
			golog.Errorf("Unable to delete case notifications: %s", err)
			return err
		}

		// insert a notification into the patient case if the doctor activates a treatment plan
		if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
			PatientCaseId:    ev.Message.CaseID,
			NotificationType: CNTreatmentPlan,
			UID:              fmt.Sprintf("%s:%d", CNTreatmentPlan, ev.TreatmentPlan.Id.Int64()),
			Data: &treatmentPlanNotification{
				MessageId:       ev.Message.ID,
				DoctorId:        ev.DoctorId,
				TreatmentPlanId: ev.TreatmentPlan.Id.Int64(),
				CaseId:          ev.Message.CaseID,
			},
		}); err != nil {
			golog.Errorf("Unable to insert notification item for case: %s", err)
			return err
		}

		patient := ev.Patient
		var err error
		if patient == nil {
			patient, err = dataAPI.GetPatientFromId(ev.PatientId)
			if err != nil {
				golog.Errorf("unable to get patient from id: %s", err)
				return err
			}
		}

		// notify patient of new treatment plan
		if err := notificationManager.NotifyPatient(patient, ev); err != nil {
			golog.Errorf("Unable to notify patient: %s", err)
			return err
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *patient_visit.VisitStartedEvent) error {
		if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
			PatientCaseId:    ev.PatientCaseId,
			NotificationType: CNIncompleteVisit,
			UID:              CNIncompleteVisit,
			Data: &incompleteVisitNotification{
				PatientVisitId: ev.VisitId,
			},
		}); err != nil {
			golog.Errorf("Unable to insert notification item for case: %s", err)
			return err
		}

		return nil

	})

	dispatch.Default.Subscribe(func(ev *patient_visit.VisitSubmittedEvent) error {

		// delete the notification that indicates that the user still has to complete
		// the visit
		if err := dataAPI.DeleteCaseNotification(CNIncompleteVisit, ev.PatientCaseId); err != nil {
			golog.Errorf("Unable to delete case notifications: %s", err)
			return err
		}

		if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
			PatientCaseId:    ev.PatientCaseId,
			NotificationType: CNVisitSubmitted,
			UID:              CNVisitSubmitted,
			Data:             &visitSubmittedNotification{},
		}); err != nil {
			golog.Errorf("Unable to insert notification item for case: %s", err)
			return err
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *app_event.AppEvent) error {

		// act on this event if it represents a patient having viewed a treatment plan
		if ev.Resource == "treatment_plan" && ev.Role == api.PATIENT_ROLE && ev.Action == app_event.ViewedAction {

			patientId, err := dataAPI.GetPatientIdFromAccountId(ev.AccountId)
			if err != nil {
				golog.Errorf("unable to get patient id from account id: %s", err)
				return err
			}

			treatmentPlan, err := dataAPI.GetTreatmentPlanForPatient(patientId, ev.ResourceId)
			if err != nil {
				golog.Errorf("Unable to get treatment plan for patient: %s", err)
				return err
			}

			if err := dataAPI.DeleteCaseNotification(fmt.Sprintf("%s:%d", CNTreatmentPlan, treatmentPlan.Id.Int64()), treatmentPlan.PatientCaseId.Int64()); err != nil {
				golog.Errorf("Unable to delete case notification: %s", err)
				return err
			}
		}

		// act on the event if it represents a patient having viewed a message
		if ev.Resource == "case_message" && ev.Role == api.PATIENT_ROLE && ev.Action == app_event.ViewedAction {
			caseID, err := dataAPI.GetCaseIDFromMessageID(ev.ResourceId)
			if err != nil {
				golog.Errorf("Unable to get case id from message id: %s", err)
				return err
			}

			if err := dataAPI.DeleteCaseNotification(fmt.Sprintf("%s:%d", CNMessage, ev.ResourceId), caseID); err != nil {
				golog.Errorf("Unable to delete case notification: %s", err)
				return err
			}
		}

		return nil
	})

}

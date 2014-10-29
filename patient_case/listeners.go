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
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/schedmsg"
)

const (
	treatmentPlanViewedEvent = "treatment_plan_viewed"
)

type treatmentPlanViewedContext struct {
	PatientFirstName         string
	ProviderShortDisplayName string
}

func init() {
	schedmsg.MustRegisterEvent(treatmentPlanViewedEvent)
}

func InitListeners(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, notificationManager *notify.NotificationManager) {
	dispatcher.Subscribe(func(ev *messages.PostEvent) error {

		// delete any pending visit submitted notifications for case
		if err := dataAPI.DeleteCaseNotification(CNVisitSubmitted, ev.Case.Id.Int64()); err != nil {
			golog.Errorf("Unable to delete case notifications: %s", err)
			return err
		}

		// insert notification into patient case if the doctor or ma
		// sent the patient a message
		if ev.Person.RoleType == api.DOCTOR_ROLE || ev.Person.RoleType == api.MA_ROLE {
			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseId:    ev.Case.Id.Int64(),
				NotificationType: CNMessage,
				UID:              fmt.Sprintf("%s:%d", CNMessage, ev.Message.ID),
				Data: &messageNotification{
					MessageId: ev.Message.ID,
					DoctorId:  ev.Person.Doctor.DoctorId.Int64(),
					CaseId:    ev.Message.CaseID,
					Role:      ev.Person.RoleType,
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

	dispatcher.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {

		// delete any pending visit submitted notifications for case
		if err := dataAPI.DeleteCaseNotification(CNVisitSubmitted, ev.TreatmentPlan.PatientCaseId.Int64()); err != nil {
			golog.Errorf("Unable to delete case notifications: %s", err)
			return err
		}

		isRevisedTreatmentPlan, err := dataAPI.IsRevisedTreatmentPlan(ev.TreatmentPlan.Id.Int64())
		if err != nil {
			golog.Errorf(err.Error())
			return err
		}

		if isRevisedTreatmentPlan {
			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseId:    ev.TreatmentPlan.PatientCaseId.Int64(),
				NotificationType: CNMessage,
				UID:              fmt.Sprintf("%s:%d", CNMessage, ev.Message.ID),
				Data: &messageNotification{
					MessageId: ev.Message.ID,
					DoctorId:  ev.DoctorId,
					CaseId:    ev.Message.CaseID,
					Role:      api.DOCTOR_ROLE,
				},
			}); err != nil {
				golog.Errorf("Unable to insert notification item for case: %s", err)
				return err
			}
		} else {
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
		}

		patient := ev.Patient
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

	dispatcher.Subscribe(func(ev *patient.VisitStartedEvent) error {

		isFollowup, err := dataAPI.IsFollowupVisit(ev.VisitId)
		if err != nil {
			golog.Errorf(err.Error())
			return err
		}

		if isFollowup {
			if err := dataAPI.DeleteCaseNotification(CNStartFollowup, ev.PatientCaseId); err != nil {
				golog.Errorf("Unable to delete case notifications: %s", err)
				return err
			}

			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseId:    ev.PatientCaseId,
				NotificationType: CNIncompleteFollowup,
				UID:              CNIncompleteVisit,
				Data: &incompleteFollowupVisitNotification{
					PatientVisitID: ev.VisitId,
					CaseID:         ev.PatientCaseId,
				},
			}); err != nil {
				golog.Errorf("Unable to insert notification item for case: %s", err)
				return err
			}
		} else {
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
		}
		return nil

	})

	dispatcher.Subscribe(func(ev *patient.VisitSubmittedEvent) error {

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
			Data: &visitSubmittedNotification{
				CaseID: ev.PatientCaseId,
			},
		}); err != nil {
			golog.Errorf("Unable to insert notification item for case: %s", err)
			return err
		}

		return nil
	})

	dispatcher.Subscribe(func(ev *app_event.AppEvent) error {

		// act on this event if it represents a patient having viewed a treatment plan
		if ev.Resource == "treatment_plan" && ev.Role == api.PATIENT_ROLE && ev.Action == app_event.ViewedAction {

			if ev.ResourceId == 0 {
				return nil
			}

			patient, err := dataAPI.GetPatientFromAccountId(ev.AccountId)
			if err != nil {
				golog.Errorf("Unable to get patient: %s", err)
				return err
			}

			treatmentPlan, err := dataAPI.GetTreatmentPlanForPatient(patient.PatientId.Int64(), ev.ResourceId)
			if err == api.NoRowsError {
				golog.Warningf("Treatment plan %d doesnt exist", ev.ResourceId)
				return nil
			} else if err != nil {
				golog.Errorf("Unable to get treatment plan for patient: %s", err)
				return err
			}

			if err := dataAPI.DeleteCaseNotification(fmt.Sprintf("%s:%d", CNTreatmentPlan, treatmentPlan.Id.Int64()), treatmentPlan.PatientCaseId.Int64()); err != nil {
				golog.Errorf("Unable to delete case notification: %s", err)
				return err
			}

			maAssignment, err := dataAPI.GetActiveCareTeamMemberForCase(api.MA_ROLE, treatmentPlan.PatientCaseId.Int64())
			if err != nil {
				golog.Infof("Unable to get ma in the care team: %s", err)
				return err
			}

			ma, err := dataAPI.GetDoctorFromId(maAssignment.ProviderID)
			if err != nil {
				golog.Errorf("Unable to get ma: %s", err)
				return err
			}

			if err := schedmsg.ScheduleInAppMessage(
				dataAPI,
				treatmentPlanViewedEvent,
				&treatmentPlanViewedContext{
					PatientFirstName:         patient.FirstName,
					ProviderShortDisplayName: ma.ShortDisplayName,
				},
				&schedmsg.CaseInfo{
					PatientID:     patient.PatientId.Int64(),
					PatientCaseID: treatmentPlan.PatientCaseId.Int64(),
					SenderRole:    api.MA_ROLE,
					ProviderID:    ma.DoctorId.Int64(),
					PersonID:      ma.PersonId,
				},
			); err != nil {
				golog.Errorf("Unable to schedule in app message: %s", err)
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

			// if there exists a pending followup visit then go ahead and insert a notification
			// reason that we insert a pending followup notification on the read of a message is
			// to avoid competing CTAs on the patient side when there is a followup message attached to a message.
			pendingFollowupVisit, err := dataAPI.PendingFollowupVisitForCase(caseID)
			if err != api.NoRowsError && err != nil {
				golog.Errorf(err.Error())
				return err
			}
			if pendingFollowupVisit != nil {
				if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
					PatientCaseId:    caseID,
					NotificationType: CNStartFollowup,
					UID:              CNStartFollowup,
					Data: &startFollowupVisitNotification{
						PatientVisitID: pendingFollowupVisit.PatientVisitId.Int64(),
						CaseID:         caseID,
					},
				}); err != nil {
					golog.Errorf("Unable to insert notification item for case: %s", err)
					return err
				}
			}
		}

		return nil
	})

}

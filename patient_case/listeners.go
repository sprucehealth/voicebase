package patient_case

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/appevent"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/schedmsg"
)

const (
	treatmentPlanViewedEvent                = "treatment_plan_viewed"
	notifyTreatmentPlanCreatedEmailType     = "notify-treatment-plan-created"
	notifyNewMessageEmailType               = "notify-new-message"
	notifyParentalConsentCompletedEmailType = "minor-no-push-parent-consent-confirmation"
	txtParentalConsentCompletedNotification = "parental_consent_completed_notification"
)

type treatmentPlanViewedContext struct {
	PatientFirstName         string
	ProviderShortDisplayName string
}

// PatientNotifier can send push notifications to a patient.
type PatientNotifier interface {
	NotifyPatient(patient *common.Patient, msg *notify.Message) error
}

func init() {
	schedmsg.MustRegisterEvent(treatmentPlanViewedEvent)
}

// InitListeners subscribes to dispatched events.
func InitListeners(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, notificationManager PatientNotifier) {
	dispatcher.Subscribe(func(ev *messages.PostEvent) error {

		// delete any pending visit submitted notifications for case
		if err := dataAPI.DeleteCaseNotification(CNVisitSubmitted, ev.Case.ID.Int64()); err != nil {
			golog.Errorf("Unable to delete case notifications: %s", err)
			return err
		}

		// 1:
		// insert notification into patient case if the doctor or ma
		// sent the patient a message
		if ev.Person.RoleType == api.RoleDoctor || ev.Person.RoleType == api.RoleCC {
			uid := fmt.Sprintf("%s:%d", CNMessage, ev.Message.ID)
			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseID:    ev.Case.ID.Int64(),
				NotificationType: CNMessage,
				UID:              uid,
				Data: &messageNotification{
					MessageID: ev.Message.ID,
					DoctorID:  ev.Person.Doctor.ID.Int64(),
					CaseID:    ev.Message.CaseID,
					Role:      ev.Person.RoleType,
				},
			}); err != nil {
				golog.Errorf("Unable to insert notification item for case: %s", err)
				return err
			}

			patient, err := dataAPI.GetPatientFromID(ev.Case.PatientID)
			if err != nil {
				golog.Errorf("Unable to get patient from id: %s", err)
				return err
			}

			// notify the patient of the message
			if err := notificationManager.NotifyPatient(
				patient, &notify.Message{
					ShortMessage: "You have a new message on Spruce.",
					EmailType:    notifyNewMessageEmailType,
					PushID:       uid,
				}); err != nil {
				golog.Errorf("Unable to notify patient: %s", err)
				return err
			}
		}

		// 2:
		// If the doctor has messaged the patient make sure we reassign to the patient's CC
		if ev.Person.RoleType == api.RoleDoctor {
			cc, err := dataAPI.GetActiveCareTeamMemberForCase(api.RoleCC, ev.Message.CaseID)
			if err != nil {
				golog.Errorf("Unable to locate care coordinator for patient: %s", err)
				return err
			}

			if cc != nil {
				ccDoctor, err := dataAPI.GetDoctorFromID(cc.ProviderID)
				if err != nil {
					return err
				}

				dispatcher.Publish(&messages.CaseAssignEvent{
					Message: ev.Message,
					Person:  ev.Person,
					Case:    ev.Case,
					Doctor:  ev.Person.Doctor,
					MA:      ccDoctor,
				})
			}
		}
		return nil
	})

	dispatcher.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {

		// delete any pending visit submitted notifications for case
		if err := dataAPI.DeleteCaseNotification(CNVisitSubmitted, ev.TreatmentPlan.PatientCaseID.Int64()); err != nil {
			golog.Errorf("Unable to delete case notifications: %s", err)
			return err
		}

		isRevisedTreatmentPlan, err := dataAPI.IsRevisedTreatmentPlan(ev.TreatmentPlan.ID.Int64())
		if err != nil {
			golog.Errorf(err.Error())
			return err
		}

		var uid string
		if isRevisedTreatmentPlan {
			uid = fmt.Sprintf("%s:%d", CNMessage, ev.Message.ID)
			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseID:    ev.TreatmentPlan.PatientCaseID.Int64(),
				NotificationType: CNMessage,
				UID:              uid,
				Data: &messageNotification{
					MessageID: ev.Message.ID,
					DoctorID:  ev.DoctorID,
					CaseID:    ev.Message.CaseID,
					Role:      api.RoleDoctor,
				},
			}); err != nil {
				golog.Errorf("Unable to insert notification item for case: %s", err)
				return err
			}
		} else {
			uid = fmt.Sprintf("%s:%d", CNTreatmentPlan, ev.TreatmentPlan.ID.Int64())
			// insert a notification into the patient case if the doctor activates a treatment plan
			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseID:    ev.Message.CaseID,
				NotificationType: CNTreatmentPlan,
				UID:              uid,
				Data: &treatmentPlanNotification{
					MessageID:       ev.Message.ID,
					DoctorID:        ev.DoctorID,
					TreatmentPlanID: ev.TreatmentPlan.ID.Int64(),
					CaseID:          ev.Message.CaseID,
				},
			}); err != nil {
				golog.Errorf("Unable to insert notification item for case: %s", err)
				return err
			}
		}

		patient := ev.Patient
		if patient == nil {
			patient, err = dataAPI.GetPatientFromID(ev.PatientID)
			if err != nil {
				golog.Errorf("unable to get patient from id: %s", err)
				return err
			}
		}

		// notify patient of new treatment plan
		if err := notificationManager.NotifyPatient(
			patient,
			&notify.Message{
				PushID:       uid,
				ShortMessage: "Your doctor has reviewed your case.",
				EmailType:    notifyTreatmentPlanCreatedEmailType,
			}); err != nil {
			golog.Errorf("Unable to notify patient: %s", err)
			return err
		}

		return nil
	})

	dispatcher.Subscribe(func(ev *patient.VisitStartedEvent) error {

		visit, err := dataAPI.GetPatientVisitFromID(ev.VisitID)
		if err != nil {
			golog.Errorf(err.Error())
			return err
		}

		if visit.IsFollowup {
			if err := dataAPI.DeleteCaseNotification(CNStartFollowup, ev.PatientCaseID); err != nil {
				golog.Errorf("Unable to delete case notifications: %s", err)
				return err
			}

			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseID:    ev.PatientCaseID,
				NotificationType: CNIncompleteFollowup,
				UID:              CNIncompleteVisit,
				Data: &incompleteFollowupVisitNotification{
					PatientVisitID: ev.VisitID,
					CaseID:         ev.PatientCaseID,
				},
			}); err != nil {
				golog.Errorf("Unable to insert notification item for case: %s", err)
				return err
			}
		} else {
			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseID:    ev.PatientCaseID,
				NotificationType: CNIncompleteVisit,
				UID:              CNIncompleteVisit,
				Data: &incompleteVisitNotification{
					PatientVisitID: ev.VisitID,
				},
			}); err != nil {
				golog.Errorf("Unable to insert notification item for case: %s", err)
				return err
			}
		}
		return nil

	})

	dispatcher.Subscribe(func(ev *patient.VisitSubmittedEvent) error {

		// update the case from OPEN->ACTIVE if the case is currently considered open
		activeStatus := common.PCStatusActive
		if err := dataAPI.UpdatePatientCase(ev.PatientCaseID, &api.PatientCaseUpdate{
			Status: &activeStatus,
		}); err != nil {
			golog.Errorf("Unable to update status of case from open->active")
			return err
		}

		// delete the notification that indicates that the user still has to complete
		// the visit
		if err := dataAPI.DeleteCaseNotification(CNIncompleteVisit, ev.PatientCaseID); err != nil {
			golog.Errorf("Unable to delete case notifications: %s", err)
			return err
		}

		if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
			PatientCaseID:    ev.PatientCaseID,
			NotificationType: CNVisitSubmitted,
			UID:              CNVisitSubmitted,
			Data: &visitSubmittedNotification{
				CaseID:  ev.PatientCaseID,
				VisitID: ev.VisitID,
			},
		}); err != nil {
			golog.Errorf("Unable to insert notification item for case: %s", err)
			return err
		}

		return nil
	})

	dispatcher.Subscribe(func(ev *patient_visit.PreSubmissionVisitTriageEvent) error {

		if err := dataAPI.DeleteCaseNotification(CNIncompleteVisit, ev.CaseID); err != nil {
			golog.Errorf("Unable to delete case notification: %s", err.Error())
			return err
		}

		if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
			PatientCaseID:    ev.CaseID,
			NotificationType: CNPreSubmissionTriage,
			UID:              fmt.Sprintf("%s:%d", CNPreSubmissionTriage, ev.VisitID),
			Data: &preSubmissionTriageNotification{
				CaseID:        ev.CaseID,
				VisitID:       ev.VisitID,
				ActionMessage: ev.ActionMessage,
				Title:         ev.Title,
				ActionURL:     ev.ActionURL,
			},
		}); err != nil {
			golog.Errorf("Unable to insert notification item for case: %s", err.Error())
			return err
		}

		return nil
	})

	dispatcher.Subscribe(func(ev *appevent.AppEvent) error {

		// act on this event if it represents a patient having viewed a treatment plan
		if ev.Resource == "treatment_plan" && ev.Role == api.RolePatient && ev.Action == appevent.ViewedAction {

			if ev.ResourceID == 0 {
				return nil
			}

			patient, err := dataAPI.GetPatientFromAccountID(ev.AccountID)
			if err != nil {
				golog.Errorf("Unable to get patient: %s", err)
				return err
			}

			treatmentPlan, err := dataAPI.GetTreatmentPlanForPatient(patient.ID, ev.ResourceID)
			if api.IsErrNotFound(err) {
				golog.Warningf("Treatment plan %d doesnt exist", ev.ResourceID)
				return nil
			} else if err != nil {
				golog.Errorf("Unable to get treatment plan for patient: %s", err)
				return err
			}

			// mark the treatment plan as being viewed
			if !treatmentPlan.PatientViewed {
				treatmentPlan.PatientViewed = true
				if err := dataAPI.UpdateTreatmentPlan(treatmentPlan.ID.Int64(), &api.TreatmentPlanUpdate{
					PatientViewed: &treatmentPlan.PatientViewed,
				}); err != nil {
					golog.Errorf("Unable to update treatment plan for patient: %s", err.Error())
				}
			}

			if err := dataAPI.DeleteCaseNotification(fmt.Sprintf("%s:%d", CNTreatmentPlan, treatmentPlan.ID.Int64()), treatmentPlan.PatientCaseID.Int64()); err != nil {
				golog.Errorf("Unable to delete case notification: %s", err)
				return err
			}

			maAssignment, err := dataAPI.GetActiveCareTeamMemberForCase(api.RoleCC, treatmentPlan.PatientCaseID.Int64())
			if err != nil {
				golog.Infof("Unable to get ma in the care team: %s", err)
				return err
			}

			ma, err := dataAPI.GetDoctorFromID(maAssignment.ProviderID)
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
					PatientID:     patient.ID,
					PatientCaseID: treatmentPlan.PatientCaseID.Int64(),
					SenderRole:    api.RoleCC,
					ProviderID:    ma.ID.Int64(),
					PersonID:      ma.PersonID,
				},
			); err != nil {
				golog.Errorf("Unable to schedule in app message: %s", err)
				return err
			}
		}

		// act on the event if it represents a patient having viewed a message
		if ev.Resource == "case_message" && ev.Role == api.RolePatient && ev.Action == appevent.ViewedAction {

			// nothing to do if the resourceID is not present
			if ev.ResourceID == 0 {
				return nil
			}

			caseID, err := dataAPI.GetCaseIDFromMessageID(ev.ResourceID)
			if err != nil {
				golog.Errorf("Unable to get case id from message id %d for account id %d: %s", ev.ResourceID, ev.AccountID, err)
				return err
			}

			if err := dataAPI.DeleteCaseNotification(fmt.Sprintf("%s:%d", CNMessage, ev.ResourceID), caseID); err != nil {
				golog.Errorf("Unable to delete case notification: %s", err)
				return err
			}

			// if there exists a pending followup visit then go ahead and insert a notification
			// reason that we insert a pending followup notification on the read of a message is
			// to avoid competing CTAs on the patient side when there is a followup message attached to a message.
			pendingFollowupVisit, err := dataAPI.PendingFollowupVisitForCase(caseID)
			if !api.IsErrNotFound(err) && err != nil {
				golog.Errorf(err.Error())
				return err
			}
			if pendingFollowupVisit != nil {
				if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
					PatientCaseID:    caseID,
					NotificationType: CNStartFollowup,
					UID:              CNStartFollowup,
					Data: &startFollowupVisitNotification{
						PatientVisitID: pendingFollowupVisit.ID.Int64(),
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

	// Notify a minor patient when their parent completes the consent flow
	dispatcher.SubscribeAsync(func(ev *patient.ParentalConsentCompletedEvent) error {
		text, err := dataAPI.LocalizedText(api.LanguageIDEnglish, []string{txtParentalConsentCompletedNotification})
		if err != nil {
			golog.Errorf("Failed to get localized text: %s", err)
			return nil
		}
		patient, err := dataAPI.Patient(ev.ChildPatientID, true)
		if err != nil {
			golog.Errorf("Failed to get patient: %s", err)
			return err
		}
		msg := &notify.Message{
			ShortMessage: text[txtParentalConsentCompletedNotification],
			EmailType:    notifyParentalConsentCompletedEmailType,
			PushID:       fmt.Sprintf("%s:%s", CNVisitAuthorized, ev.ChildPatientID),
		}
		if err := notificationManager.NotifyPatient(patient, msg); err != nil {
			golog.Errorf("Failed to notify patient: %s", err)
			return err
		}
		return nil
	})
}

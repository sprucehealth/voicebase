package homelog

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_url"
	"carefront/common"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"carefront/messages"
	patientApiService "carefront/patient"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

func InitListeners(dataAPI api.DataAPI) {
	dispatch.Default.Subscribe(func(ev *apiservice.VisitStartedEvent) error {
		// Insert an incomplete notification when a patient starts a visit
		_, err := dataAPI.InsertPatientNotification(ev.PatientId, &common.Notification{
			UID:             incompleteVisit,
			Dismissible:     false,
			DismissOnAction: false,
			Priority:        1000,
			Data: &incompleteVisitNotification{
				VisitId: ev.VisitId,
			},
		})
		return err
	})

	dispatch.Default.Subscribe(func(ev *apiservice.VisitSubmittedEvent) error {
		// Remove the incomplete visit notification when the patient submits a visit
		if err := dataAPI.DeletePatientNotificationByUID(ev.PatientId, incompleteVisit); err != nil {
			golog.Errorf("Failed to remove incomplete visit notification for patient %d: %s", ev.PatientId, err.Error())
		}

		doctor, err := dataAPI.GetDoctorFromId(ev.DoctorId)
		if err != nil {
			return err
		}

		// Add "visit submitted" to health log
		params := url.Values{}
		params.Set("visit_id", strconv.FormatInt(ev.VisitId, 10))
		if _, err := dataAPI.InsertOrUpdatePatientHealthLogItem(ev.PatientId, &common.HealthLogItem{
			UID: fmt.Sprintf("visit_submitted:%d", ev.VisitId),
			Data: &titledLogItem{
				Title:    "Visit Submitted",
				Subtitle: fmt.Sprintf("With Dr. %s", doctor.LastName),
				IconURL:  app_url.GetSpruceAssetUrl(app_url.IconHomeVisitNormal),
				TapURL:   app_url.GetSpruceActionUrl(app_url.ViewPatientVisitAction, params),
			},
		}); err != nil {
			golog.Errorf("Failed to insert visit submitted into health log for patient %d: %s", ev.PatientId, err.Error())
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *apiservice.VisitReviewSubmittedEvent) error {
		doctor, err := dataAPI.GetDoctorFromId(ev.DoctorId)
		if err != nil {
			return err
		}

		// Add "treatment plan created" notification
		if _, err := dataAPI.InsertPatientNotification(ev.PatientId, &common.Notification{
			UID:             visitReviewed,
			Dismissible:     true,
			DismissOnAction: true,
			Priority:        1000,
			Data: &visitReviewedNotification{
				VisitId:  ev.VisitId,
				DoctorId: doctor.DoctorId.Int64(),
			},
		}); err != nil {
			golog.Errorf("Failed to insert treatment plan created into noficiation queue for patient %d: %s", ev.PatientId, err.Error())
		}

		// Add "treatment plan created" to health log
		params := url.Values{}
		params.Set("treatment_plan_id", strconv.FormatInt(ev.TreatmentPlanId, 10))
		if _, err := dataAPI.InsertOrUpdatePatientHealthLogItem(ev.PatientId, &common.HealthLogItem{
			UID: fmt.Sprintf("treatment_plan_created:%d", ev.TreatmentPlanId),
			Data: &titledLogItem{
				Title:    "Treatment Plan",
				Subtitle: fmt.Sprintf("Created By. %s", doctor.LastName),
				IconURL:  app_url.GetSpruceAssetUrl(app_url.IconHomeTreatmentPlanNormal),
				TapURL:   app_url.GetSpruceActionUrl(app_url.ViewTreatmentPlanAction, params),
			},
		}); err != nil {
			golog.Errorf("Failed to insert visit treatment plan created into health log for patient %d: %s", ev.PatientId, err.Error())
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *patientApiService.CareTeamAssingmentEvent) error {
		for _, a := range ev.Assignments {
			if a.ProviderRole == api.DOCTOR_ROLE {
				doctor, err := dataAPI.GetDoctorFromId(a.ProviderId)
				if err != nil {
					golog.Errorf("Failed to lookup doctor %d: %s", a.ProviderId, err.Error())
				} else {
					if _, err := dataAPI.InsertOrUpdatePatientHealthLogItem(ev.PatientId, &common.HealthLogItem{
						UID: fmt.Sprintf("doctor_added:%d", a.ProviderId),
						Data: &textLogItem{
							Text:    fmt.Sprintf("%s %s, M.D., added to your care team.", doctor.FirstName, doctor.LastName),
							IconURL: doctor.SmallThumbnailUrl,
							TapURL:  app_url.GetSpruceActionUrl(app_url.ViewCareTeam, nil),
						},
					}); err != nil {
						golog.Errorf("Failed to insert visit treatment plan created into health log for patient %d: %s", ev.PatientId, err.Error())
					}
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
		from := people[ev.FromId]
		if from == nil {
			return errors.New("failed to find person conversation is from")
		}
		to := people[ev.ToId]
		if to == nil {
			return errors.New("failed to find person conversation is addressed to")
		}

		// Insert health log item for patient
		var doctorPerson *common.Person
		var patientPerson *common.Person
		for _, p := range people {
			switch p.RoleType {
			case api.PATIENT_ROLE:
				patientPerson = p
			case api.DOCTOR_ROLE:
				doctorPerson = p
			}
		}
		if doctorPerson != nil && patientPerson != nil {
			params := url.Values{}
			params.Set("conversation_id", strconv.FormatInt(ev.ConversationId, 10))
			if _, err := dataAPI.InsertOrUpdatePatientHealthLogItem(patientPerson.RoleId, &common.HealthLogItem{
				UID: fmt.Sprintf("conversation:%d", ev.ConversationId),
				Data: &titledLogItem{
					Title:    fmt.Sprintf("Conversation with Dr. %s", doctorPerson.Doctor.LastName),
					Subtitle: fmt.Sprintf("1 message"),
					IconURL:  app_url.GetSpruceAssetUrl(app_url.IconHomeConversationNormal),
					TapURL:   app_url.GetSpruceActionUrl(app_url.ViewMessagesAction, params),
				},
			}); err != nil {
				golog.Errorf("Failed to insert conversation item into health log for patient %d: %s", patientPerson.RoleId, err.Error())
			}
		}

		// Only notify the patient if the conversation is doctor->patient
		if to.RoleType == api.PATIENT_ROLE && from.RoleType == api.DOCTOR_ROLE {
			_, err = dataAPI.InsertPatientNotification(to.RoleId, &common.Notification{
				UID:             fmt.Sprintf("conversation:%d", ev.ConversationId),
				Dismissible:     true,
				DismissOnAction: true,
				Priority:        1000,
				Data: &newConversationNotification{
					ConversationId: ev.ConversationId,
					DoctorId:       from.RoleId,
				},
			})
			return err
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

		// Update health log item for patient
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
		if doctorPerson != nil && patientPerson != nil {
			params := url.Values{}
			params.Set("conversation_id", strconv.FormatInt(ev.ConversationId, 10))
			if _, err := dataAPI.InsertOrUpdatePatientHealthLogItem(patientPerson.RoleId, &common.HealthLogItem{
				UID: fmt.Sprintf("conversation:%d", ev.ConversationId),
				Data: &titledLogItem{
					Title:    fmt.Sprintf("Conversation with Dr. %s", doctorPerson.Doctor.LastName),
					Subtitle: fmt.Sprintf("%d messages", con.MessageCount),
					IconURL:  app_url.GetSpruceAssetUrl(app_url.IconLogMessage),
					TapURL:   app_url.GetSpruceActionUrl(app_url.ViewMessagesAction, params),
				},
			}); err != nil {
				golog.Errorf("Failed to insert conversation item into health log for patient %d: %s", patientPerson.RoleId, err.Error())
			}
		}

		// Only notify the patient if the reply is doctor->patient
		if from.RoleType == api.DOCTOR_ROLE && patientPerson != nil {
			_, err = dataAPI.InsertPatientNotification(patientPerson.RoleId, &common.Notification{
				UID:             fmt.Sprintf("conversation:%d", ev.ConversationId),
				Dismissible:     true,
				DismissOnAction: true,
				Priority:        1000,
				Data: &conversationReplyNotification{
					ConversationId: ev.ConversationId,
					DoctorId:       from.RoleId,
				},
			})
			return err
		}
		return nil
	})
}

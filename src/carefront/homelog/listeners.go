package homelog

import (
	"carefront/api"
	"carefront/app_url"
	"carefront/common"
	"carefront/doctor_treatment_plan"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"carefront/notify"
	patientApiService "carefront/patient"
	"carefront/patient_visit"
	"fmt"
)

func InitListeners(dataAPI api.DataAPI, notificationManager *notify.NotificationManager) {
	dispatch.Default.Subscribe(func(ev *patient_visit.VisitStartedEvent) error {
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

	dispatch.Default.Subscribe(func(ev *patient_visit.VisitSubmittedEvent) error {
		// Remove the incomplete visit notification when the patient submits a visit
		if err := dataAPI.DeletePatientNotificationByUID(ev.PatientId, incompleteVisit); err != nil {
			golog.Errorf("Failed to remove incomplete visit notification for patient %d: %s", ev.PatientId, err.Error())
		}

		doctor, err := dataAPI.GetDoctorFromId(ev.DoctorId)
		if err != nil {
			return err
		}

		// Add "visit submitted" to health log
		if _, err := dataAPI.InsertOrUpdatePatientHealthLogItem(ev.PatientId, &common.HealthLogItem{
			UID: fmt.Sprintf("visit_submitted:%d", ev.VisitId),
			Data: &titledLogItem{
				Title:    "Visit Submitted",
				Subtitle: fmt.Sprintf("With Dr. %s", doctor.LastName),
				IconURL:  app_url.IconHomeVisitNormal,
				TapURL:   app_url.ViewPatientVisitAction(ev.VisitId),
			},
		}); err != nil {
			golog.Errorf("Failed to insert visit submitted into health log for patient %d: %s", ev.PatientId, err.Error())
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanCreatedEvent) error {
		doctor, err := dataAPI.GetDoctorFromId(ev.DoctorId)
		if err != nil {
			return err
		}

		// Remove the any previous treatment plan created notifications
		if err := dataAPI.DeletePatientNotificationByUID(ev.PatientId, treatmentPlanCreated); err != nil {
			golog.Errorf("Failed to remove treatment plan created notification for patient %d: %s", ev.PatientId, err.Error())
		}

		// Add "treatment plan created" notification
		if _, err := dataAPI.InsertPatientNotification(ev.PatientId, &common.Notification{
			UID:             treatmentPlanCreated,
			Dismissible:     true,
			DismissOnAction: true,
			Priority:        1000,
			Data: &treatmentPlanCreatedNotification{
				VisitId:         ev.VisitId,
				DoctorId:        doctor.DoctorId.Int64(),
				TreatmentPlanId: ev.TreatmentPlanId,
			},
		}); err != nil {
			golog.Errorf("Failed to insert treatment plan created into noficiation queue for patient %d: %s", ev.PatientId, err.Error())
		}

		// Notify Patient
		patient := ev.Patient
		if ev.Patient == nil {
			patient, err = dataAPI.GetPatientFromId(ev.PatientId)
			if err != nil {
				golog.Errorf("Unable to get patient from id: %s", err)
				return err
			}

		}

		if err := notificationManager.NotifyPatient(patient, ev); err != nil {
			golog.Errorf("Unable to notify patient: %s", err)
			return err
		}

		// Add "treatment plan created" to health log
		if _, err := dataAPI.InsertOrUpdatePatientHealthLogItem(ev.PatientId, &common.HealthLogItem{
			UID: fmt.Sprintf("treatment_plan_created:%d", ev.TreatmentPlanId),
			Data: &titledLogItem{
				Title:    "Treatment Plan",
				Subtitle: fmt.Sprintf("Created By. %s", doctor.LastName),
				IconURL:  app_url.IconHomeTreatmentPlanNormal,
				TapURL:   app_url.ViewTreatmentPlanAction(ev.TreatmentPlanId),
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
							TapURL:  app_url.ViewCareTeam(),
						},
					}); err != nil {
						golog.Errorf("Failed to insert visit treatment plan created into health log for patient %d: %s", ev.PatientId, err.Error())
					}
				}
			}
		}
		return nil
	})
}

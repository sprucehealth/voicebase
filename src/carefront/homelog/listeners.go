package homelog

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"fmt"
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
		if _, err := dataAPI.InsertOrUpdatePatientHealthLogItem(ev.PatientId, &common.HealthLogItem{
			UID: fmt.Sprintf("visit_submitted:%d", ev.VisitId),
			Data: &titledLogItem{
				Title:    "Visit Submitted",
				Subtitle: fmt.Sprintf("With Dr. %s", doctor.LastName),
				IconURL:  "spruce:///image/icon_log_visit",
				TapURL:   fmt.Sprintf("spruce:///action/view_visit/visit_id=%d", ev.VisitId),
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

		planID, err := dataAPI.GetActiveTreatmentPlanForPatientVisit(ev.DoctorId, ev.VisitId)
		if err != nil {
			return err
		}

		// Add "treatment plan created" to health log
		if _, err := dataAPI.InsertOrUpdatePatientHealthLogItem(ev.PatientId, &common.HealthLogItem{
			UID: fmt.Sprintf("treatment_plan_created:%d", planID),
			Data: &titledLogItem{
				Title:    "Treatment Plan",
				Subtitle: fmt.Sprintf("Created By. %s", doctor.LastName),
				IconURL:  "spruce:///image/icon_log_treatment_plan",
				TapURL:   fmt.Sprintf("spruce:///action/view_treatment_plan?treatment_plan_id=%d", planID),
			},
		}); err != nil {
			golog.Errorf("Failed to insert visit treatment plan created into health log for patient %d: %s", ev.PatientId, err.Error())
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *apiservice.CareTeamAssingmentEvent) error {
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
							IconURL: fmt.Sprintf("spruce:///image/thumbnail_care_team_%d", doctor.DoctorId.Int64()), // TODO
							TapURL:  "spruce:///action/view_care_team",
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

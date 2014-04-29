package homelog

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/dispatch"
)

func InitListeners(dataAPI api.DataAPI) {
	// Insert an incomplete notification when a patient starts a visit
	dispatch.Default.Subscribe(func(ev *apiservice.VisitStartedEvent) error {
		_, err := dataAPI.InsertPatientNotification(ev.PatientId, &common.Notification{
			UID:             incompleteVisit,
			Dismissible:     false,
			DismissOnAction: false,
			Priority:        1000,
			Data: &IncompleteVisitNotification{
				VisitId: ev.VisitId,
			},
		})
		return err
	})

	// Remove the incomplete visit notification when the patient submits a visit
	dispatch.Default.Subscribe(func(ev *apiservice.VisitSubmittedEvent) error {
		return dataAPI.DeletePatientNotificationByUID(ev.PatientId, incompleteVisit)
	})
}

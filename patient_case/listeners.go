package patient_case

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/treatment_plan"
)

func InitListeners(dataAPI api.DataAPI) {
	dispatch.Default.Subscribe(func(ev *messages.PostEvent) error {

		// insert notification into patient case if the doctor
		// sent the patient a message
		if ev.Person.RoleType == api.DOCTOR_ROLE {
			if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
				PatientCaseId:    ev.Case.Id.Int64(),
				NotificationType: common.CNMessage,
				ItemId:           ev.Message.ID,
				Data: &messageNotificationData{
					MessageId:    ev.Message.ID,
					DoctorId:     ev.Person.Doctor.DoctorId.Int64(),
					DismissOnTap: true,
				},
			}); err != nil {
				golog.Errorf("Unable to insert notification item for case: %s", err)
				return err
			}
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {
		// insert a notification into the patient case if the doctor activates a treatment plan
		if err := dataAPI.InsertCaseNotification(&common.CaseNotification{
			PatientCaseId:    ev.Message.CaseID,
			NotificationType: common.CNTreatmentPlan,
			ItemId:           ev.TreatmentPlanId,
			Data: &treatmentPlanNotificationData{
				MessageId:       ev.Message.ID,
				DoctorId:        ev.DoctorId,
				TreatmentPlanId: ev.TreatmentPlanId,
			},
		}); err != nil {
			golog.Errorf("Unable to insert notification item for case: %s", err)
			return err
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *treatment_plan.TreatmentPlanOpenedEvent) error {
		// if treatment plan is opened by patient, delete any case notifications pertaining
		// to treatment plan
		if ev.RoleType == api.PATIENT_ROLE {
			if err := dataAPI.DeleteCaseNotification(ev.TreatmentPlan.PatientCaseId.Int64(), ev.TreatmentPlan.Id.Int64(), common.CNTreatmentPlan); err != nil {
				golog.Errorf("Unable to delete case notification: %s", err)
				return err
			}
		}

		return nil
	})
}

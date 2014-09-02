package schedmsg

import (
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient_visit"
)

func InitListeners(scheduledMsgQueue *common.SQSQueue, dataAPI api.DataAPI) {
	dispatch.Default.Subscribe(func(ev *patient_visit.VisitChargedEvent) error {
		return scheduleInAppMessageFromTemplate(dataAPI, scheduledMsgQueue,
			smVisitChargedEventType,
			ev.PatientID,
			ev.PatientCaseID)
	})

	dispatch.Default.Subscribe(func(ev *app_event.AppEvent) error {
		// act on this event if it represents a patient having viewed a treatment plan
		if ev.Resource == "treatment_plan" && ev.Role == api.PATIENT_ROLE && ev.Action == app_event.ViewedAction {
			patientId, err := dataAPI.GetPatientIdFromAccountId(ev.AccountId)
			if err != nil {
				return err
			}

			treatmentPlan, err := dataAPI.GetTreatmentPlanForPatient(patientId, ev.ResourceId)
			if err != nil {
				return err
			}
			return scheduleInAppMessageFromTemplate(dataAPI, scheduledMsgQueue,
				smTreatmentPlanViewedType,
				treatmentPlan.PatientId.Int64(),
				treatmentPlan.PatientCaseId.Int64())
		}
		return nil
	})
}

func scheduleInAppMessageFromTemplate(dataAPI api.DataAPI, schedMsgQueue *common.SQSQueue, event string, patientID, patientCaseID int64) error {
	// look up any existing templates
	templates, err := dataAPI.ScheduledMessageTemplates(event, scheduledMsgTypes)
	if err == api.NoRowsError {
		// nothing to do for this event if no templates exist
		return nil
	} else if err != nil {
		return err
	}

	// create a scheduled message and enqeue a job for every template
	for _, template := range templates {

		caseMessageTemplate := template.AppMessageJSON.(*caseMessage)

		assignment, err := dataAPI.GetActiveCareTeamMemberForCase(caseMessageTemplate.SenderRole, patientCaseID)
		if err != nil {
			golog.Errorf("Unable to get care team member: %s", err)
			return err
		}

		doctor, err := dataAPI.GetDoctorFromId(assignment.ProviderID)
		if err != nil {
			golog.Errorf("Unable to get care provider from id: %s", err)
			return err
		}

		patient, err := dataAPI.GetPatientFromId(patientID)
		if err != nil {
			golog.Errorf("Unable to get patient from id: %s", err)
			return err
		}

		scheduledMessage := &common.ScheduledMessage{
			Type:        event,
			PatientID:   patientID,
			MessageType: common.SMCaseMessageType,
			MessageJSON: &caseMessage{
				Message:        fillInTags(caseMessageTemplate.Message, patient, doctor),
				PatientCaseID:  patientCaseID,
				SenderPersonID: doctor.PersonId,
				SenderRole:     caseMessageTemplate.SenderRole,
				ProviderID:     doctor.DoctorId.Int64(),
			},
			Scheduled: time.Now().Add(time.Duration(template.SchedulePeriod) * time.Second),
			Status:    common.SMScheduled,
		}

		if err := dataAPI.CreateScheduledMessage(scheduledMessage); err != nil {
			golog.Errorf("Unable to create scheduled message: %s", err)
			return err
		}

		if err := apiservice.QueueUpJob(schedMsgQueue, &schedSQSMessage{
			ScheduledMessageID: scheduledMessage.ID,
			ScheduledTime:      scheduledMessage.Scheduled,
		}); err != nil {
			golog.Errorf("Unable to queue up msg: %s", err)
			return err
		}
	}
	return nil
}

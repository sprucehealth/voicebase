package schedmsg

import (
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient_visit"
)

func InitListeners(dataAPI api.DataAPI) {
	dispatch.Default.Subscribe(func(ev *patient_visit.InsuredPatientEvent) error {
		go func() {
			if err := scheduleInAppMessageFromTemplate(dataAPI,
				common.SMInsuredPatientEvent,
				ev.PatientID,
				ev.PatientCaseID); err != nil {
				golog.Errorf(err.Error())
			}
		}()
		return nil
	})

	dispatch.Default.Subscribe(func(ev *patient_visit.UninsuredPatientEvent) error {
		go func() {
			if err := scheduleInAppMessageFromTemplate(dataAPI,
				common.SMUninsuredPatientEvent,
				ev.PatientID,
				ev.PatientCaseID); err != nil {
				golog.Errorf(err.Error())
			}
		}()
		return nil
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
			return scheduleInAppMessageFromTemplate(dataAPI,
				common.SMTreatmentPlanViewedEvent,
				treatmentPlan.PatientId.Int64(),
				treatmentPlan.PatientCaseId.Int64())
		}
		return nil
	})
}

func scheduleInAppMessageFromTemplate(dataAPI api.DataAPI, event common.ScheduledMessageEvent, patientID, patientCaseID int64) error {

	// look up any existing templates
	templates, err := dataAPI.ScheduledMessageTemplates(event)
	if err == api.NoRowsError {
		// nothing to do for this event if no templates exist
		return nil
	} else if err != nil {
		return err
	}

	// create a scheduled message and enqeue a job for every template
	for _, template := range templates {

		assignment, err := dataAPI.GetActiveCareTeamMemberForCase(api.MA_ROLE, patientCaseID)
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
			Event:       event,
			PatientID:   patientID,
			MessageType: common.SMCaseMessageType,
			MessageJSON: &caseMessage{
				Message:        fillInTags(template.Message, patient, doctor),
				PatientCaseID:  patientCaseID,
				SenderPersonID: doctor.PersonId,
				SenderRole:     api.MA_ROLE,
				ProviderID:     doctor.DoctorId.Int64(),
			},
			Scheduled: time.Now().Add(time.Duration(template.SchedulePeriod) * time.Second),
			Status:    common.SMScheduled,
		}

		if err := dataAPI.CreateScheduledMessage(scheduledMessage); err != nil {
			golog.Errorf("Unable to create scheduled message: %s", err)
			return err
		}
	}
	return nil
}

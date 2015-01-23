package doctor_treatment_plan

import (
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/schedmsg"
)

const (
	checkTreatments = iota
	checkRegimenPlan
	checkNote

	treatmentPlanScheduledMessageEvent = "treatment_plan"
)

func init() {
	schedmsg.MustRegisterEvent(treatmentPlanScheduledMessageEvent)
}

func InitListeners(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) {
	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies any section of the treatment plan
	dispatcher.Subscribe(func(ev *TreatmentPlanUpdatedEvent) error {
		return markTPDeviatedIfContentChanged(
			ev.TreatmentPlanID,
			ev.DoctorID,
			dataAPI,
			ev.SectionUpdated)
	})

	dispatcher.Subscribe(func(ev *TreatmentPlanSubmittedEvent) error {
		// Create a scheduled message for every message scheduled in the treatment plan
		msgs, err := dataAPI.ListTreatmentPlanScheduledMessages(ev.TreatmentPlan.ID.Int64())
		if err != nil {
			return err
		}
		now := time.Now()
		for _, m := range msgs {
			// Should always be nil in this case because the treatment plan can only be submitted once,
			// but it's probably good just to make sure to avoid duplicate messages.
			if m.ScheduledMessageID != nil {
				continue
			}

			id, err := dataAPI.CreateScheduledMessage(&common.ScheduledMessage{
				Event:     treatmentPlanScheduledMessageEvent,
				PatientID: ev.TreatmentPlan.PatientID,
				Message: &schedmsg.TreatmentPlanMessage{
					MessageID:       m.ID,
					TreatmentPlanID: ev.TreatmentPlan.ID.Int64(),
				},
				Created:   now,
				Scheduled: now.Add(24 * time.Hour * time.Duration(m.ScheduledDays)),
				Status:    common.SMScheduled,
			})
			if err != nil {
				golog.Errorf("Failed to create scheduled message for %d: %d %s", m.ID, ev.TreatmentPlan.ID.Int64(), err.Error())
			} else if err := dataAPI.UpdateTreatmentPlanScheduledMessage(m.ID, &id); err != nil {
				golog.Errorf("Failed to update scheduled message %d: %s", m.ID, err.Error())
			}
		}
		return nil
	})
}

func markTPDeviatedIfContentChanged(treatmentPlanID, doctorID int64, dataAPI api.DataAPI, sectionToCheck Sections) error {
	doctorTreatmentPlan, err := dataAPI.GetAbridgedTreatmentPlan(treatmentPlanID, doctorID)
	if err != nil {
		return err
	}

	// nothing to do here if the content source doesn't exist or has already deviated from the source
	if doctorTreatmentPlan.ContentSource == nil || doctorTreatmentPlan.ContentSource.HasDeviated {
		return nil
	}

	if sectionToCheck&ScheduledMessagesSection == ScheduledMessagesSection {
		return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanID)
	}

	if sectionToCheck&ResourceGuidesSection == ResourceGuidesSection {
		return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanID)
	}

	var regimenPlanToCompare *common.RegimenPlan
	var treatmentsToCompare *common.TreatmentList
	if sectionToCheck&TreatmentsSection == TreatmentsSection || sectionToCheck&RegimenSection == RegimenSection {
		switch doctorTreatmentPlan.ContentSource.Type {
		case common.TPContentSourceTypeFTP:
			// get favorite treatment plan to compare
			favoriteTreatmentPlan, err := dataAPI.FavoriteTreatmentPlan(doctorTreatmentPlan.ContentSource.ID.Int64())
			if err != nil {
				return err
			}

			regimenPlanToCompare = favoriteTreatmentPlan.RegimenPlan
			treatmentsToCompare = favoriteTreatmentPlan.TreatmentList

		case common.TPContentSourceTypeTreatmentPlan:
			// get parent treatment plan to compare
			parentTreatmentPlan, err := dataAPI.GetTreatmentPlan(doctorTreatmentPlan.Parent.ParentID.Int64(), doctorID)
			if err != nil {
				return err
			}

			regimenPlanToCompare = parentTreatmentPlan.RegimenPlan
			treatmentsToCompare = parentTreatmentPlan.TreatmentList
		}
	}

	if sectionToCheck&TreatmentsSection == TreatmentsSection {
		treatments, err := dataAPI.GetTreatmentsBasedOnTreatmentPlanID(doctorTreatmentPlan.ID.Int64())
		if err != nil {
			return err
		}
		if !treatmentsToCompare.Equals(&common.TreatmentList{Treatments: treatments}) {
			return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanID)
		}
	}

	if sectionToCheck&RegimenSection == RegimenSection {
		regimenPlan, err := dataAPI.GetRegimenPlanForTreatmentPlan(treatmentPlanID)
		if err != nil {
			return err
		}
		if !regimenPlanToCompare.Equals(regimenPlan) {
			return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanID)
		}
	}

	if sectionToCheck&NoteSection == NoteSection {
		switch doctorTreatmentPlan.ContentSource.Type {
		case common.TPContentSourceTypeFTP:
			ftp, err := dataAPI.FavoriteTreatmentPlan(doctorTreatmentPlan.ContentSource.ID.Int64())
			if err != nil {
				return err
			}
			note, err := dataAPI.GetTreatmentPlanNote(treatmentPlanID)
			if err != nil {
				return err
			}
			if ftp.Note != note {
				return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanID)
			}
		case common.TPContentSourceTypeTreatmentPlan:
			note1, err := dataAPI.GetTreatmentPlanNote(doctorTreatmentPlan.ContentSource.ID.Int64())
			if err != nil {
				return err
			}
			note2, err := dataAPI.GetTreatmentPlanNote(treatmentPlanID)
			if err != nil {
				return err
			}

			if note1 != note2 {
				return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanID)
			}
		}
	}

	return nil
}

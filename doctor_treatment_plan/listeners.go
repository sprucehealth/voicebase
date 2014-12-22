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
	// favorite treatment if the doctor modifies the treatments for the treatment plan
	dispatcher.Subscribe(func(ev *TreatmentsAddedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanID, ev.DoctorID, dataAPI, checkTreatments)
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the regimen section
	dispatcher.Subscribe(func(ev *RegimenPlanAddedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanID, ev.DoctorID, dataAPI, checkRegimenPlan)
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the personalized note
	dispatcher.Subscribe(func(ev *TreatmentPlanNoteUpdatedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanID, ev.DoctorID, dataAPI, checkNote)
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
				golog.Errorf("Failed to create scheduled message for %d: %s", m.ID, ev.TreatmentPlan.ID.Int64(), err.Error())
			} else if err := dataAPI.UpdateTreatmentPlanScheduledMessage(m.ID, &id); err != nil {
				golog.Errorf("Failed to update scheduled message %d: %s", m.ID, err.Error())
			}
		}
		return nil
	})

	dispatcher.Subscribe(func(ev *TreatmentPlanScheduledMessagesUpdatedEvent) error {
		return dataAPI.MarkTPDeviatedFromContentSource(ev.TreatmentPlanID)
	})

	dispatcher.Subscribe(func(ev *TreatmentPlanResourceGuidesUpdatedEvent) error {
		return dataAPI.MarkTPDeviatedFromContentSource(ev.TreatmentPlanID)
	})
}

func markTPDeviatedIfContentChanged(treatmentPlanID, doctorID int64, dataAPI api.DataAPI, sectionToCheck int) error {
	doctorTreatmentPlan, err := dataAPI.GetAbridgedTreatmentPlan(treatmentPlanID, doctorID)
	if err != nil {
		return err
	}

	// nothing to do here if the content source doesn't exist or has already deviated from the source
	if doctorTreatmentPlan.ContentSource == nil || doctorTreatmentPlan.ContentSource.HasDeviated {
		return nil
	}

	var regimenPlanToCompare *common.RegimenPlan
	var treatmentsToCompare *common.TreatmentList

	if sectionToCheck != checkNote {
		switch doctorTreatmentPlan.ContentSource.Type {
		case common.TPContentSourceTypeFTP:
			// get favorite treatment plan to compare
			favoriteTreatmentPlan, err := dataAPI.GetFavoriteTreatmentPlan(doctorTreatmentPlan.ContentSource.ID.Int64())
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

	switch sectionToCheck {
	case checkTreatments:
		treatments, err := dataAPI.GetTreatmentsBasedOnTreatmentPlanID(doctorTreatmentPlan.ID.Int64())
		if err != nil {
			return err
		}

		if !treatmentsToCompare.Equals(&common.TreatmentList{Treatments: treatments}) {
			return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanID)
		}
	case checkRegimenPlan:
		regimenPlan, err := dataAPI.GetRegimenPlanForTreatmentPlan(treatmentPlanID)
		if err != nil {
			return err
		}

		if !regimenPlanToCompare.Equals(regimenPlan) {
			return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanID)
		}
	case checkNote:
		switch doctorTreatmentPlan.ContentSource.Type {
		case common.TPContentSourceTypeFTP:
			ftp, err := dataAPI.GetFavoriteTreatmentPlan(doctorTreatmentPlan.ContentSource.ID.Int64())
			if err != nil {
				return err
			}
			note, err := dataAPI.GetTreatmentPlanNote(treatmentPlanID)
			if err != nil {
				return err
			}
			if note != ftp.Note {
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

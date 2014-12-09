package doctor_treatment_plan

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
)

const (
	checkTreatments  = "treatments"
	checkRegimenPlan = "regimenPlan"
	checkNote        = "note"
)

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
}

func markTPDeviatedIfContentChanged(treatmentPlanID, doctorID int64, dataAPI api.DataAPI, sectionToCheck string) error {
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

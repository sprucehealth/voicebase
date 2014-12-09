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
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanID, ev.DoctorId, dataAPI, checkTreatments)
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the regimen section
	dispatcher.Subscribe(func(ev *RegimenPlanAddedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanID, ev.DoctorId, dataAPI, checkRegimenPlan)
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the personalized note
	dispatcher.Subscribe(func(ev *TreatmentPlanNoteUpdatedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanID, ev.DoctorID, dataAPI, checkNote)
	})
}

func markTPDeviatedIfContentChanged(treatmentPlanId, doctorId int64, dataAPI api.DataAPI, sectionToCheck string) error {
	doctorTreatmentPlan, err := dataAPI.GetAbridgedTreatmentPlan(treatmentPlanId, doctorId)
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
			parentTreatmentPlan, err := dataAPI.GetTreatmentPlan(doctorTreatmentPlan.Parent.ParentId.Int64(), doctorId)
			if err != nil {
				return err
			}

			regimenPlanToCompare = parentTreatmentPlan.RegimenPlan
			treatmentsToCompare = parentTreatmentPlan.TreatmentList
		}
	}

	switch sectionToCheck {
	case checkTreatments:
		treatments, err := dataAPI.GetTreatmentsBasedOnTreatmentPlanId(doctorTreatmentPlan.Id.Int64())
		if err != nil {
			return err
		}

		if !treatmentsToCompare.Equals(&common.TreatmentList{Treatments: treatments}) {
			return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanId)
		}
	case checkRegimenPlan:
		regimenPlan, err := dataAPI.GetRegimenPlanForTreatmentPlan(treatmentPlanId)
		if err != nil {
			return err
		}

		if !regimenPlanToCompare.Equals(regimenPlan) {
			return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanId)
		}
	case checkNote:
		switch doctorTreatmentPlan.ContentSource.Type {
		case common.TPContentSourceTypeFTP:
			ftp, err := dataAPI.GetFavoriteTreatmentPlan(doctorTreatmentPlan.ContentSource.ID.Int64())
			if err != nil {
				return err
			}
			note, err := dataAPI.GetTreatmentPlanNote(treatmentPlanId)
			if err != nil {
				return err
			}
			if note != ftp.Note {
				return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanId)
			}
		case common.TPContentSourceTypeTreatmentPlan:
			note1, err := dataAPI.GetTreatmentPlanNote(doctorTreatmentPlan.ContentSource.ID.Int64())
			if err != nil {
				return err
			}
			note2, err := dataAPI.GetTreatmentPlanNote(treatmentPlanId)
			if err != nil {
				return err
			}
			if note1 != note2 {
				return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanId)
			}
		}
	}

	return nil
}

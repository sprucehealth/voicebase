package doctor_treatment_plan

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
)

const (
	checkTreatments  = "treatments"
	checkRegimenPlan = "regimenPlan"
	checkAdvice      = "advice"
)

func InitListeners(dataAPI api.DataAPI) {
	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the treatments for the treatment plan
	dispatch.Default.Subscribe(func(ev *TreatmentsAddedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanId, ev.DoctorId, dataAPI, checkTreatments)
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the regimen section
	dispatch.Default.Subscribe(func(ev *RegimenPlanAddedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanId, ev.DoctorId, dataAPI, checkRegimenPlan)
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the advice section
	dispatch.Default.Subscribe(func(ev *AdviceAddedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanId, ev.DoctorId, dataAPI, checkAdvice)
	})

	// If the doctor successfully submits a treatment plan for an unclaimed case, then the message is saved in the message between the
	// patient and the care team. It is no longer a draft, and can be deleted.
	dispatch.Default.Subscribe(func(ev *TreatmentPlanActivatedEvent) error {
		go dataAPI.DeleteTreatmentPlanMessage(ev.DoctorId, ev.TreatmentPlan.Id.Int64())
		return nil
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
	var adviceToCompare *common.Advice
	switch doctorTreatmentPlan.ContentSource.ContentSourceType {

	case common.TPContentSourceTypeFTP:
		// get favorite treatment plan to compare
		favoriteTreatmentPlan, err := dataAPI.GetFavoriteTreatmentPlan(doctorTreatmentPlan.ContentSource.ContentSourceId.Int64())
		if err != nil {
			return err
		}

		regimenPlanToCompare = favoriteTreatmentPlan.RegimenPlan
		treatmentsToCompare = favoriteTreatmentPlan.TreatmentList
		adviceToCompare = favoriteTreatmentPlan.Advice

	case common.TPContentSourceTypeTreatmentPlan:
		// get parent treatment plan to compare
		parentTreatmentPlan, err := dataAPI.GetTreatmentPlan(doctorTreatmentPlan.Parent.ParentId.Int64(), doctorId)
		if err != nil {
			return err
		}

		regimenPlanToCompare = parentTreatmentPlan.RegimenPlan
		treatmentsToCompare = parentTreatmentPlan.TreatmentList
		adviceToCompare = parentTreatmentPlan.Advice
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
	case checkAdvice:
		advice, err := dataAPI.GetAdvicePointsForTreatmentPlan(treatmentPlanId)
		if err != nil {
			return err
		}

		if !adviceToCompare.Equals(&common.Advice{SelectedAdvicePoints: advice}) {
			return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanId)
		}
	}

	return nil
}

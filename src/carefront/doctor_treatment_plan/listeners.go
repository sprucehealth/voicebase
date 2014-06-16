package doctor_treatment_plan

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/dispatch"
	"carefront/patient_visit"
)

func InitListeners(dataAPI api.DataAPI) {

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the treatments for the treatment plan
	dispatch.Default.Subscribe(func(ev *TreatmentsAddedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanId, ev.DoctorId, dataAPI)
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the regimen section
	dispatch.Default.Subscribe(func(ev *RegimenPlanAddedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanId, ev.DoctorId, dataAPI)
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the advice section
	dispatch.Default.Subscribe(func(ev *AdviceAddedEvent) error {
		return markTPDeviatedIfContentChanged(ev.TreatmentPlanId, ev.DoctorId, dataAPI)
	})

	dispatch.Default.Subscribe(func(ev *patient_visit.DiagnosisModifiedEvent) error {
		return updateDiagnosisSummary(dataAPI, ev.DoctorId, ev.PatientVisitId, ev.TreatmentPlanId)
	})

	dispatch.Default.Subscribe(func(ev *NewTreatmentPlanStartedEvent) error {
		return updateDiagnosisSummary(dataAPI, ev.DoctorId, ev.PatientVisitId, ev.TreatmentPlanId)
	})

}

func markTPDeviatedIfContentChanged(treatmentPlanId, doctorId int64, dataAPI api.DataAPI) error {
	doctorTreatmentPlan, err := dataAPI.GetTreatmentPlan(treatmentPlanId, doctorId)
	if err != nil {
		return err
	}

	// nothing to do here if the content source doesn't exist or has already deviated from the source
	if doctorTreatmentPlan.ContentSource == nil || doctorTreatmentPlan.ContentSource.HasDeviated {
		return nil
	}

	switch doctorTreatmentPlan.ContentSource.ContentSourceType {

	case common.TPContentSourceTypeFTP:
		// get favorite treatment plan to compare
		favoriteTreatmentPlan, err := dataAPI.GetFavoriteTreatmentPlan(doctorTreatmentPlan.ContentSource.ContentSourceId.Int64())
		if err != nil {
			return err
		}

		// compare the treatment plan to the favorite treatment plan and mark as deviated if they are unequal
		if !favoriteTreatmentPlan.EqualsDoctorTreatmentPlan(doctorTreatmentPlan) {
			return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanId)
		}

	case common.TPContentSourceTypeTreatmentPlan:
		// get parent treatment plan to compare
		parentTreatmentPlan, err := dataAPI.GetTreatmentPlan(doctorTreatmentPlan.Parent.ParentId.Int64(), doctorId)
		if err != nil {
			return err
		}

		// mark the treatment plan has having deviated if the content is no longer the same as the parent
		if !parentTreatmentPlan.Equals(doctorTreatmentPlan) {
			return dataAPI.MarkTPDeviatedFromContentSource(doctorTreatmentPlan.Id.Int64())
		}
	}

	return nil
}

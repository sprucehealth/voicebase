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
	doctorTreatmentPlan, err := dataAPI.GetAbridgedTreatmentPlan(treatmentPlanId, doctorId)
	if err != nil {
		return err
	}

	// nothing to do here if the treatment plan is not linked to a favorite treatment plan or if the content has already deviated from the source
	if doctorTreatmentPlan.ContentSource == nil || doctorTreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP || doctorTreatmentPlan.ContentSource.HasDeviated {
		return nil
	}

	// get favorite treatment plan to compare
	favoriteTreatmentPlan, err := dataAPI.GetFavoriteTreatmentPlan(doctorTreatmentPlan.ContentSource.ContentSourceId.Int64())
	if err != nil {
		return err
	}

	// compare the treatment plan to the favorite treatment plan and mark as deviated if they are unequal
	if !favoriteTreatmentPlan.EqualsDoctorTreatmentPlan(doctorTreatmentPlan) {
		return dataAPI.MarkTPDeviatedFromContentSource(treatmentPlanId)
	}
	return nil
}

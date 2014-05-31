package treatment_plan

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/dispatch"
	"carefront/visit"
)

func InitListeners(dataAPI api.DataAPI) {

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the treatments for the treatment plan
	dispatch.Default.Subscribe(func(ev *apiservice.TreatmentsAddedEvent) error {
		doctorTreatmentPlan, err := dataAPI.GetAbridgedTreatmentPlan(ev.TreatmentPlanId, ev.DoctorId)
		if err != nil {
			return err
		}

		// nothing to do here if the treatment plan is not linked to a favorite treatment plan
		if doctorTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64() == 0 {
			return nil
		}

		// get the treatments from the favorite treatment plan to compare
		favoritedTreatments, err := dataAPI.GetTreatmentsInFavoriteTreatmentPlan(doctorTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64())
		if err != nil {
			return err
		}

		// compare the treatments and if a difference is found, invalidate the linkage between treatment plan
		// and favorite treatment plan
		favoritedTreatmentList := &common.TreatmentList{Treatments: favoritedTreatments}
		addedTreatmentList := &common.TreatmentList{Treatments: ev.Treatments}

		if !favoritedTreatmentList.Equals(addedTreatmentList) {
			return dataAPI.DeleteFavoriteTreatmentPlanMapping(doctorTreatmentPlan.Id.Int64(),
				doctorTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64())
		}

		return nil
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the regimen section
	dispatch.Default.Subscribe(func(ev *apiservice.RegimenPlanAddedEvent) error {
		doctorTreatmentPlan, err := dataAPI.GetAbridgedTreatmentPlan(ev.TreatmentPlanId, ev.DoctorId)
		if err != nil {
			return err
		}

		// nothing to do here if the treatment plan is not linked to a favorite treatment plan
		if doctorTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64() == 0 {
			return nil
		}

		// get the treatments from the favorite treatment plan to compare
		favoritedRegimenPlan, err := dataAPI.GetRegimenPlanInFavoriteTreatmentPlan(doctorTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64())
		if err != nil {
			return err
		}

		// compare the regimen plans and if they are unequal delete the mapping
		if !favoritedRegimenPlan.Equals(ev.RegimenPlan) {
			return dataAPI.DeleteFavoriteTreatmentPlanMapping(doctorTreatmentPlan.Id.Int64(),
				doctorTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64())
		}
		return nil
	})

	// subscribe to invalidate the link between a treatment plan and
	// favorite treatment if the doctor modifies the advice section
	dispatch.Default.Subscribe(func(ev *apiservice.AdviceAddedEvent) error {
		doctorTreatmentPlan, err := dataAPI.GetAbridgedTreatmentPlan(ev.TreatmentPlanId, ev.DoctorId)
		if err != nil {
			return err
		}

		// nothing to do here if the treatment plan is not linked to a favorite treatment plan
		if doctorTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64() == 0 {
			return nil
		}

		// get the treatments from the favorite treatment plan to compare
		favoritedAdvice, err := dataAPI.GetAdviceInFavoriteTreatmentPlan(doctorTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64())
		if err != nil {
			return err
		}

		// compare the regimen plans and if they are unequal delete the mapping
		if !favoritedAdvice.Equals(ev.Advice) {
			return dataAPI.DeleteFavoriteTreatmentPlanMapping(doctorTreatmentPlan.Id.Int64(),
				doctorTreatmentPlan.DoctorFavoriteTreatmentPlanId.Int64())
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *visit.DiagnosisModifiedEvent) error {
		return updateDiagnosisSummary(dataAPI, ev.DoctorId, ev.PatientVisitId, ev.TreatmentPlanId)
	})

	dispatch.Default.Subscribe(func(ev *NewTreatmentPlanStartedEvent) error {
		return updateDiagnosisSummary(dataAPI, ev.DoctorId, ev.PatientVisitId, ev.TreatmentPlanId)
	})

}

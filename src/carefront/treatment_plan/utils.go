package treatment_plan

import (
	"carefront/api"
	"carefront/common"
	"fmt"
)

func populateTreatmentPlan(dataApi api.DataAPI, patientVisitId int64, treatmentPlanId int64) (*common.TreatmentPlan, error) {
	treatmentPlan := &common.TreatmentPlan{}

	var err error
	treatmentPlan.DiagnosisSummary, err = dataApi.GetDiagnosisSummaryForTreatmentPlan(treatmentPlanId)
	if err != nil {
		return nil, fmt.Errorf("Unable to get diagnosis summary for patient visit: %s", err)
	}

	treatmentPlan.TreatmentList = &common.TreatmentList{}
	treatmentPlan.TreatmentList.Treatments, err = dataApi.GetTreatmentsBasedOnTreatmentPlanId(treatmentPlanId)
	if err != nil {
		return nil, fmt.Errorf("Unable to get treatment plan for this patient visit id: %s", err)
	}

	treatmentPlan.RegimenPlan, err = dataApi.GetRegimenPlanForTreatmentPlan(treatmentPlanId)
	if err != nil {
		return nil, fmt.Errorf("Unable to get regimen plan for this patient visit id: %s", err)
	}

	treatmentPlan.Followup, err = dataApi.GetFollowUpTimeForTreatmentPlan(treatmentPlanId)
	if err != nil {
		return nil, fmt.Errorf("Unable to get follow up information for this patient visit: %s", err)
	}

	advicePoints, err := dataApi.GetAdvicePointsForTreatmentPlan(treatmentPlanId)
	if err != nil {
		return nil, fmt.Errorf("Unable to get advice for patient visit: %s", err)
	}

	if advicePoints != nil && len(advicePoints) > 0 {
		treatmentPlan.Advice = &common.Advice{
			SelectedAdvicePoints: advicePoints,
		}
	}

	return treatmentPlan, nil
}

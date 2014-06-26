package treatment_plan

import (
	"carefront/api"
	"carefront/common"
	"fmt"
)

func populateTreatmentPlan(dataApi api.DataAPI, patientVisitId int64, treatmentPlan *common.TreatmentPlan) error {

	var err error
	treatmentPlan.TreatmentList = &common.TreatmentList{}
	treatmentPlan.TreatmentList.Treatments, err = dataApi.GetTreatmentsBasedOnTreatmentPlanId(treatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get treatment plan for this patient visit id: %s", err)
	}

	treatmentPlan.RegimenPlan, err = dataApi.GetRegimenPlanForTreatmentPlan(treatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get regimen plan for this patient visit id: %s", err)
	}

	advicePoints, err := dataApi.GetAdvicePointsForTreatmentPlan(treatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get advice for patient visit: %s", err)
	}

	if advicePoints != nil && len(advicePoints) > 0 {
		treatmentPlan.Advice = &common.Advice{
			SelectedAdvicePoints: advicePoints,
		}
	}

	return nil
}

package treatment_plan

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
)

func populateTreatmentPlan(dataApi api.DataAPI, treatmentPlan *common.TreatmentPlan) error {

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

	return nil
}

func generateViewsForTreatments(treatmentPlan *common.TreatmentPlan, doctor *common.Doctor, dataAPI api.DataAPI, forMedicationsTab bool) []tpView {
	views := make([]tpView, 0)
	if treatmentPlan.TreatmentList != nil {
		for _, treatment := range treatmentPlan.TreatmentList.Treatments {
			iconURL := app_url.PrescriptionIcon(treatment.DrugRoute)
			if treatment.OTC {
				iconURL = app_url.IconOTCLarge
			}

			pView := &tpPrescriptionView{
				Title:       fmt.Sprintf("%s %s %s", treatment.DrugName, treatment.DosageStrength, treatment.DrugForm),
				Description: treatment.PatientInstructions,
				IconURL:     iconURL,
				IconWidth:   50,
				IconHeight:  50,
			}

			if forMedicationsTab {
				pView.Subtitle = "Prescribed on <timestamp>"
				pView.SubtitleHasTokens = true
				pView.Timestamp = treatment.CreationDate
			}

			views = append(views, &tpCardView{
				Views: []tpView{pView},
			})

			if forMedicationsTab {
				pView.Buttons = append(pView.Buttons, &tpPrescriptionButtonView{
					Text:    "Treatment Plan",
					IconURL: app_url.IconTreatmentPlanBlueButton,
					TapURL:  app_url.ViewTreatmentPlanAction(treatmentPlan.Id.Int64()),
				})
			}

			// only add button if treatment guide exists
			if ndc := treatment.DrugDBIds[erx.NDC]; ndc != "" {
				if exists, err := dataAPI.DoesDrugDetailsExist(ndc); exists {
					pView.Buttons = append(pView.Buttons, &tpPrescriptionButtonView{
						Text:    "Prescription Guide",
						IconURL: app_url.IconRXGuide,
						TapURL:  app_url.ViewTreatmentGuideAction(treatment.Id.Int64()),
					})
				} else if err != nil && err != api.NoRowsError {
					golog.Errorf("Error when trying to check if drug details exist: %s", err)
				}
			}
		}
	}
	return views
}

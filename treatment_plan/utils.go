package treatment_plan

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
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

func generateViewsForTreatments(treatmentList *common.TreatmentList, doctor *common.Doctor, dataAPI api.DataAPI, forMedicationsTab bool) []tpView {
	var views []tpView
	if treatmentList != nil {
		for _, treatment := range treatmentList.Treatments {

			iconURL := app_url.IconRXLarge
			smallHeaderText := "Prescription"
			if treatment.OTC {
				iconURL = app_url.IconOTCLarge
				smallHeaderText = "Over the Counter"
			}

			if forMedicationsTab {
				smallHeaderText = fmt.Sprintf("Prescribed on %s", treatment.CreationDate.Format(apiservice.TimeFormatLayout))
			}

			pView := &tpPrescriptionView{
				Title:           fmt.Sprintf("%s %s", treatment.DrugInternalName, treatment.DosageStrength),
				Description:     treatment.PatientInstructions,
				SmallHeaderText: smallHeaderText,
				IconURL:         iconURL,
			}
			views = append(views, &tpCardView{
				Views: []tpView{pView},
			})

			// only add button if treatment guide exists
			if ndc := treatment.DrugDBIds[erx.NDC]; ndc != "" {
				if exists, err := dataAPI.DoesDrugDetailsExist(ndc); exists {
					pView.Buttons = []tpView{
						&tpPrescriptionButtonView{
							Text:    "View RX Guide",
							IconURL: app_url.IconGuide,
							TapURL:  app_url.ViewTreatmentGuideAction(treatment.Id.Int64()),
						},
					}
				} else if err != nil && err != api.NoRowsError {
					golog.Errorf("Error when trying to check if drug details exist: %s", err)
				}
			}
		}
		views = append(views, &tpButtonFooterView{
			FooterText: fmt.Sprintf("If you have any questions about your treatment plan, send Dr. %s a message.", doctor.LastName),
			ButtonText: fmt.Sprintf("Message Dr. %s", doctor.LastName),
			IconURL:    app_url.IconMessage,
			TapURL:     app_url.MessageAction(),
		})
	}
	return views
}

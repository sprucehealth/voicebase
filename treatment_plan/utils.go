package treatment_plan

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/views"
)

func populateTreatmentPlan(dataAPI api.DataAPI, treatmentPlan *common.TreatmentPlan) error {
	var err error
	treatmentPlan.TreatmentList = &common.TreatmentList{}
	treatmentPlan.TreatmentList.Treatments, err = dataAPI.GetTreatmentsBasedOnTreatmentPlanID(treatmentPlan.ID.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get treatment plan for this patient visit id: %s", err)
	}

	treatmentPlan.RegimenPlan, err = dataAPI.GetRegimenPlanForTreatmentPlan(treatmentPlan.ID.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get regimen plan for this patient visit id: %s", err)
	}

	treatmentPlan.ResourceGuides, err = dataAPI.ListTreatmentPlanResourceGuides(treatmentPlan.ID.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get resource guides for treatment plan %d: %s", treatmentPlan.ID.Int64(), err.Error())
	}

	return nil
}

func GenerateViewsForTreatments(tl *common.TreatmentList, treatmentPlanID int64, dataAPI api.DataAPI, forMedicationsTab bool) []views.View {
	tViews := make([]views.View, 0)
	if tl != nil {
		drugQueries := make([]*api.DrugDetailsQuery, len(tl.Treatments))
		for i, t := range tl.Treatments {
			drugQueries[i] = &api.DrugDetailsQuery{
				NDC:         t.DrugDBIDs[erx.NDC],
				GenericName: t.GenericDrugName,
				Route:       t.DrugRoute,
				Form:        t.DrugForm,
			}
		}
		drugDetails, err := dataAPI.MultiQueryDrugDetailIDs(drugQueries)
		if err != nil {
			// It's possible to continue. We just won't return treatment guide buttons
			golog.Errorf("Failed to query for drug details: %s", err.Error())
			// The drugDetails slice is expected to have the same number of elements as treatments
			drugDetails = make([]int64, len(tl.Treatments))
		}
		for i, treatment := range tl.Treatments {
			iconURL := app_url.IconRXLarge
			if treatment.OTC {
				iconURL = app_url.IconOTCLarge
			}

			var subtitle string
			if treatment.OTC {
				subtitle = "Over-the-counter"
			} else {
				switch treatment.DrugRoute {
				case "topical":
					subtitle = "Topical Prescription"
				case "oral":
					subtitle = "Oral Prescription"
				default:
					subtitle = "Prescription"
				}
			}
			pView := &tpPrescriptionView{
				Title:       fmt.Sprintf("%s %s %s", treatment.DrugName, treatment.DosageStrength, treatment.DrugForm),
				Subtitle:    subtitle,
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

			tViews = append(tViews, &tpCardView{
				Views: []views.View{pView},
			})

			if forMedicationsTab {
				pView.Buttons = append(pView.Buttons, &tpPrescriptionButtonView{
					Text:    "Treatment Plan",
					IconURL: app_url.IconTreatmentPlanBlueButton,
					TapURL:  app_url.ViewTreatmentPlanAction(treatmentPlanID),
				})
			}

			var tapURL *app_url.SpruceAction
			if treatment.ID.Int64() != 0 {
				tapURL = app_url.ViewTreatmentGuideAction(treatment.ID.Int64())
			} else {
				tapURL = app_url.ViewRXGuideGuideAction(treatment.GenericDrugName, treatment.DrugRoute, treatment.DrugForm, treatment.DosageStrength)
			}
			if drugDetails[i] != 0 {
				pView.Buttons = append(pView.Buttons, &tpPrescriptionButtonView{
					Text:    "Prescription Guide",
					IconURL: app_url.IconRXGuide,
					TapURL:  tapURL,
				})
			}
		}
	}
	return tViews
}

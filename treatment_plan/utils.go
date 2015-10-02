package treatment_plan

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/features"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/views"
	"golang.org/x/net/context"
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

func generateViewsForTreatmentsAndInstructions(ctx context.Context, tp *common.TreatmentPlan, patient *common.Patient, dataAPI api.DataAPI) ([]views.View, []views.View) {
	var treatmentViews, instructionViews []views.View

	// TREATMENT VIEWS
	if len(tp.TreatmentList.Treatments) > 0 {
		treatmentViews = append(treatmentViews, GenerateViewsForTreatments(ctx, tp.TreatmentList, tp.ID.Int64(), dataAPI, false)...)
		cardViews := []views.View{
			&tpCardTitleView{
				Title: "How to get your treatments",
			},
		}
		hasRX := false
		hasOTC := false
		for _, t := range tp.TreatmentList.Treatments {
			if t.OTC {
				hasOTC = true
			} else {
				hasRX = true
			}
		}
		if hasRX {
			cardViews = append(cardViews,
				&tpTextView{
					Text:  "Prescription",
					Style: views.SubheaderStyle,
				},
				&tpTextView{
					Text: "Your prescriptions have been sent to your pharmacy. We suggest calling ahead to ask about price. If it seems expensive, message your care coordinator for help.",
				},
			)
		}
		if hasOTC {
			cardViews = append(cardViews,
				&tpTextView{
					Text:  "Over-the-counter",
					Style: views.SubheaderStyle,
				},
				&tpTextView{
					Text: "Check with your pharmacist before looking for your over-the-counter treatment in the aisles. OTC treatments may be less expensive when purchased through the pharmacy.",
				},
			)
		}
		cardViews = append(cardViews,
			&tpTextView{
				Text:  "Your pharmacy",
				Style: views.SubheaderStyle,
			},
			&tpPharmacyView{
				Text:     "Your prescriptions should be ready soon. Call your pharmacy to confirm a pickup time.",
				Pharmacy: patient.Pharmacy,
			},
		)
		treatmentViews = append(treatmentViews,
			&tpCardView{
				Views: cardViews,
			},
			&tpButtonFooterView{
				FooterText:       fmt.Sprintf("If you have any questions about your treatment plan, message your care team."),
				ButtonText:       "Send a Message",
				IconURL:          app_url.IconMessage,
				TapURL:           app_url.SendCaseMessageAction(tp.PatientCaseID.Int64()),
				CenterFooterText: true,
			},
		)
	}

	// INSTRUCTION VIEWS
	if tp.RegimenPlan != nil && len(tp.RegimenPlan.Sections) > 0 {
		for _, regimenSection := range tp.RegimenPlan.Sections {
			cView := &tpCardView{
				Views: []views.View{},
			}
			instructionViews = append(instructionViews, cView)

			cView.Views = append(cView.Views, &tpCardTitleView{
				Title: regimenSection.Name,
			})

			for _, regimenStep := range regimenSection.Steps {
				cView.Views = append(cView.Views, &tpListElementView{
					ElementStyle: styleBulleted,
					Text:         regimenStep.Text,
				})
			}
		}
	}

	if len(tp.ResourceGuides) != 0 {
		rgViews := []views.View{
			&tpCardTitleView{
				Title: "Resources",
			},
		}
		for i, g := range tp.ResourceGuides {
			if i != 0 {
				rgViews = append(rgViews, &views.SmallDivider{})
			}
			rgViews = append(rgViews, &tpLargeIconTextButtonView{
				Text:       g.Title,
				IconURL:    g.PhotoURL,
				IconWidth:  66,
				IconHeight: 66,
				TapURL:     app_url.ViewResourceGuideAction(g.ID),
			})
		}
		instructionViews = append(instructionViews, &tpCardView{
			Views: rgViews,
		})
	}

	instructionViews = append(instructionViews, &tpButtonFooterView{
		FooterText:       "If you have any questions about your treatment plan, message your care team.",
		ButtonText:       "Send a Message",
		IconURL:          app_url.IconMessage,
		TapURL:           app_url.SendCaseMessageAction(tp.PatientCaseID.Int64()),
		CenterFooterText: true,
	})

	return treatmentViews, instructionViews
}

// GenerateViewsForSingleViewTreatmentPlan generates the views necessary to display the treatment plan in a single view
// format (rather than the tabbed approach we have gone with thus far).
// NOTE: While some of the work here is duplicative compared to the work in generating the multiple tab treatment plan,
// it is meant to be consolidated so that we can easily get rid of the old way of view generation for treatment plans
// in the future, and use just these views.
func GenerateViewsForSingleViewTreatmentPlan(ctx context.Context, tp *common.TreatmentPlan, pharmacy *pharmacy.PharmacyData, dataAPI api.DataAPI) []views.View {
	var singleTPViews []views.View

	// PRESCRIPTIONS
	treatmentsExist := len(tp.TreatmentList.Treatments) > 0
	if treatmentsExist {
		prescriptionCardView := &tpCardView{
			Views: []views.View{
				&tpCardTitleView{
					Title: "Treatments",
				},
			},
		}

		// Check for prescription guides for all drugs
		drugQueries := make([]*api.DrugDetailsQuery, len(tp.TreatmentList.Treatments))
		for i, t := range tp.TreatmentList.Treatments {
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
			drugDetails = make([]int64, len(tp.TreatmentList.Treatments))
		}

		for i, treatment := range tp.TreatmentList.Treatments {
			// if there are more treatments to come, add a small divider
			if i != 0 {
				prescriptionCardView.Views = append(prescriptionCardView.Views, &views.SmallDivider{})
			}

			// Subtitle
			var subtitle string
			var subtitleHasTokens bool
			switch treatment.DrugRoute {
			case "topical":
				subtitle = "Topical"
			case "oral":
				subtitle = "Oral"
			default:
				if treatment.OTC {
					subtitle = "Over the Counter"
				} else {
					subtitle = "Prescription"
				}
			}
			if treatment.NumberRefills.Int64() > 0 {
				subtitle = subtitle + " | " + fmt.Sprintf("%d refills prescribed on <short_date>", treatment.NumberRefills.Int64())
				subtitleHasTokens = true
			}

			// Icon
			iconURL := app_url.IconRXLarge
			if treatment.OTC {
				iconURL = app_url.IconOTCLarge
			}

			// Buttons
			var buttons []views.View
			if drugDetails[i] != 0 {
				var tapURL *app_url.SpruceAction
				if treatment.ID.Int64() != 0 {
					tapURL = app_url.ViewTreatmentGuideAction(treatment.ID.Int64())
				} else {
					tapURL = app_url.ViewRXGuideGuideAction(treatment.GenericDrugName, treatment.DrugRoute, treatment.DrugForm, treatment.DosageStrength)
				}
				buttons = append(buttons, &tpPrescriptionButtonView{
					Text:    "Prescription Guide",
					IconURL: app_url.IconRXGuide,
					TapURL:  tapURL,
				})
			}
			if treatment.ID.Int64() != 0 && features.CtxSet(ctx).Has(features.RXReminders) {
				buttons = append(buttons, NewPrescriptionReminderButtonView("Reminder", treatment.ID.Int64()))
			}

			prescriptionCardView.Views = append(prescriptionCardView.Views, &tpPrescriptionView{
				Title:             fullTreatmentName(treatment),
				Subtitle:          subtitle,
				SubtitleHasTokens: subtitleHasTokens,
				Timestamp:         tp.SentDate,
				PrescribedOn:      tp.SentDate.Unix(),
				Description:       treatment.PatientInstructions,
				IconURL:           iconURL,
				IconWidth:         50,
				IconHeight:        50,
				Buttons:           buttons,
			})
		}

		singleTPViews = append(singleTPViews, prescriptionCardView)
	}

	// INSTRUCTIONS
	if tp.RegimenPlan != nil && len(tp.RegimenPlan.Sections) > 0 {
		instructionsCardView := &tpCardView{
			Views: []views.View{
				&tpCardTitleView{
					Title: "Instructions",
				},
			},
		}

		for i, rs := range tp.RegimenPlan.Sections {
			if i != 0 {
				instructionsCardView.Views = append(instructionsCardView.Views, &views.SmallDivider{})
			}
			instructionsCardView.Views = append(instructionsCardView.Views, &tpTextView{
				Text:  rs.Name,
				Style: styleTitle1Medium,
			})

			for _, rt := range rs.Steps {
				instructionsCardView.Views = append(instructionsCardView.Views, &tpListElementView{
					Text:         rt.Text,
					ElementStyle: styleBulleted,
				})
			}
		}

		singleTPViews = append(singleTPViews, instructionsCardView)
	}

	// GUIDES
	if len(tp.ResourceGuides) > 0 {
		guidesCardView := &tpCardView{
			Views: []views.View{
				&tpCardTitleView{
					Title: "Skincare Guides",
				},
			},
		}

		for i, g := range tp.ResourceGuides {
			if i != 0 {
				guidesCardView.Views = append(guidesCardView.Views, &views.SmallDivider{})
			}
			guidesCardView.Views = append(guidesCardView.Views, &tpLargeIconTextButtonView{
				Text:       g.Title,
				IconURL:    g.PhotoURL,
				IconWidth:  66,
				IconHeight: 66,
				TapURL:     app_url.ViewResourceGuideAction(g.ID),
			})
		}

		singleTPViews = append(singleTPViews, guidesCardView)
	}

	// PHARMACY AND NEXT STEPS
	nextStepsCardView := &tpCardView{
		Views: []views.View{
			&tpCardTitleView{
				Title: "Next Steps",
			},
		},
	}

	if treatmentsExist {
		nextStepsCardView.Views = append(nextStepsCardView.Views,
			&tpTextView{
				Text:  "Pick up your treatments from the pharmacy.",
				Style: styleBold,
			},
			&tpSubheaderView{
				Text:  "Prescription Treatments",
				Style: styleBodyHintMedium,
			},
			&tpTextView{
				Text: "Your prescriptions have been sent to your pharmacy. We suggest calling ahead to ask about price. If it seems expensive message your care coordinator for help.",
			},
		)
	}
	nextStepsCardView.Views = append(nextStepsCardView.Views,
		&tpSubheaderView{
			Text:  "Over-the-Counter Treatments",
			Style: styleBodyHintMedium,
		},
		&tpTextView{
			Text: "Check with your pharmacist before looking for your over-the-counter treatment in the aisles. OTC treatments may be less expensive when purchased through the pharmacy.",
		},
		&tpPharmacyView{
			Pharmacy: pharmacy,
			TapURL:   app_url.ViewPharmacyInMapAction(),
		},
		&views.SmallDivider{},
		&tpTextView{
			Text:  "If you have any questions about your treatment plan, message your care team.",
			Style: styleBold,
		},
		&tpPrescriptionButtonView{
			Text:    "Message Care Team",
			IconURL: app_url.IconMessage,
			TapURL:  app_url.SendCaseMessageAction(tp.PatientCaseID.Int64()),
		},
	)

	singleTPViews = append(singleTPViews, nextStepsCardView)

	return singleTPViews
}

func GenerateViewsForTreatments(ctx context.Context, tl *common.TreatmentList, treatmentPlanID int64, dataAPI api.DataAPI, forMedicationsTab bool) []views.View {
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
				Title:       fullTreatmentName(treatment),
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

			if drugDetails[i] != 0 {
				var tapURL *app_url.SpruceAction
				if treatment.ID.Int64() != 0 {
					tapURL = app_url.ViewTreatmentGuideAction(treatment.ID.Int64())
				} else {
					tapURL = app_url.ViewRXGuideGuideAction(treatment.GenericDrugName, treatment.DrugRoute, treatment.DrugForm, treatment.DosageStrength)
				}
				pView.Buttons = append(pView.Buttons, &tpPrescriptionButtonView{
					Text:    "Prescription Guide",
					IconURL: app_url.IconRXGuide,
					TapURL:  tapURL,
				})
			}
			feat := features.CtxSet(ctx)
			if treatment.ID.Int64() != 0 && feat.Has(features.RXReminders) {
				pView.Buttons = append(pView.Buttons, NewPrescriptionReminderButtonView("Reminder", treatment.ID.Int64()))
			}
		}
	}
	return tViews
}

func fullTreatmentName(t *common.Treatment) string {
	// Filter out combinations of name + strength that lead to duplicates
	// e.g. "Doxycycline Monohydrate" + "monohydrate 100 mg"
	i1 := strings.LastIndex(t.DrugName, " ")
	i2 := strings.IndexByte(t.DosageStrength, ' ')
	if i1 > 0 && i2 > 0 {
		lastName := strings.ToLower(t.DrugName[i1+1:])
		firstStrength := strings.ToLower(t.DosageStrength[:i2])
		if lastName == firstStrength {
			return fmt.Sprintf("%s %s %s", t.DrugName[:i1], t.DosageStrength, t.DrugForm)
		}
	}
	return fmt.Sprintf("%s %s %s", t.DrugName, t.DosageStrength, t.DrugForm)
}

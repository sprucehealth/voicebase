package treatment_plan

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_url"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	question_acne_diagnosis = "q_acne_diagnosis"
	question_acne_severity  = "q_acne_severity"
	question_acne_type      = "q_acne_type"
	question_rosacea_type   = "q_acne_rosacea_type"

	diagnosedSummaryTemplateNonProd = `Dear %s,

I've taken a look at your pictures, and from what I can tell, you have %s. 

I've put together a treatment regimen for you that will take roughly 3 months to take full effect. Please stick with it as best as you can, unless you are having a concerning complications. Often times, acne gets slightly worse before it gets better.

Please keep in mind finding the right "recipe" to treat your acne may take some tweaking. As always, feel free to communicate any questions or issues you have along the way.  

Sincerely,

Dr. %s`
)

func updateDiagnosisSummary(dataApi api.DataAPI, doctorId, patientVisitId, treatmentPlanId int64) error {
	if treatmentPlanId != 0 {
		diagnosisSummary, err := dataApi.GetDiagnosisSummaryForTreatmentPlan(treatmentPlanId)
		if err != nil && err != api.NoRowsError {
			golog.Errorf("Error trying to retreive diagnosis summary for patient visit: %s", err)
		}

		if diagnosisSummary == nil || !diagnosisSummary.UpdatedByDoctor { // use what the doctor entered if the summary has been updated by the doctor
			if err = addDiagnosisSummaryForPatientVisit(dataApi, doctorId, patientVisitId, treatmentPlanId); err != nil {
				return fmt.Errorf("Something went wrong when trying to add and store the summary to the diagnosis of the patient visit: %s", err)
			}
		}
	}
	return nil
}

func addDiagnosisSummaryForPatientVisit(dataApi api.DataAPI, doctorId, patientVisitId, treatmentPlanId int64) error {
	// lookup answers for the following questions
	acneDiagnosisAnswers, err := dataApi.GetDiagnosisResponseToQuestionWithTag(question_acne_diagnosis, doctorId, patientVisitId)
	if err != nil && err != api.NoDiagnosisResponseErr {
		return err
	}

	acneSeverityAnswers, err := dataApi.GetDiagnosisResponseToQuestionWithTag(question_acne_severity, doctorId, patientVisitId)
	if err != nil && err != api.NoDiagnosisResponseErr {
		return err
	}

	acneTypeAnswers, err := dataApi.GetDiagnosisResponseToQuestionWithTag(question_acne_type, doctorId, patientVisitId)
	if err != nil && err != api.NoDiagnosisResponseErr {
		return err
	}

	rosaceaTypeAnswers, err := dataApi.GetDiagnosisResponseToQuestionWithTag(question_rosacea_type, doctorId, patientVisitId)
	if err != nil && err != api.NoDiagnosisResponseErr {
		return err
	}

	diagnosisMessage := ""
	if acneDiagnosisAnswers != nil && len(acneDiagnosisAnswers) > 0 {
		diagnosisMessage = acneDiagnosisAnswers[0].AnswerSummary
	} else {
		// nothing to do if the patient was not properly diagnosed
		return nil
	}

	// for acne vulgaris, we only want the diagnosis to indicate acne
	if (acneDiagnosisAnswers != nil && len(acneDiagnosisAnswers) > 0) && (acneSeverityAnswers != nil && len(acneSeverityAnswers) > 0) {
		if acneTypeAnswers != nil && len(acneTypeAnswers) > 0 {
			diagnosisMessage = fmt.Sprintf("%s %s %s", acneSeverityAnswers[0].AnswerSummary, joinAcneTypesIntoString(acneTypeAnswers), acneDiagnosisAnswers[0].AnswerSummary)
		} else if rosaceaTypeAnswers != nil && len(rosaceaTypeAnswers) > 0 {
			diagnosisMessage = fmt.Sprintf("%s %s %s", acneSeverityAnswers[0].AnswerSummary, joinAcneTypesIntoString(rosaceaTypeAnswers), acneDiagnosisAnswers[0].AnswerSummary)
		} else {
			diagnosisMessage = fmt.Sprintf("%s %s", acneSeverityAnswers[0].AnswerSummary, acneDiagnosisAnswers[0].PotentialAnswer)
		}
	}

	doctor, err := dataApi.GetDoctorFromId(doctorId)
	if err != nil {
		return err
	}

	patient, err := dataApi.GetPatientFromPatientVisitId(patientVisitId)
	if err != nil {
		return err
	}

	doctorFullName := fmt.Sprintf("%s %s", doctor.FirstName, doctor.LastName)

	summaryTemplate := diagnosedSummaryTemplateNonProd

	diagnosisSummary := fmt.Sprintf(summaryTemplate, strings.Title(patient.FirstName), strings.ToLower(diagnosisMessage), strings.Title(doctorFullName))
	return dataApi.AddDiagnosisSummaryForTreatmentPlan(diagnosisSummary, treatmentPlanId, doctorId)
}

func joinAcneTypesIntoString(acneTypeAnswers []*common.AnswerIntake) string {
	acneTypes := make([]string, 0)

	for _, acneTypeAnswer := range acneTypeAnswers {
		acneTypes = append(acneTypes, acneTypeAnswer.AnswerSummary)
	}

	if len(acneTypes) == 1 {
		return acneTypes[0]
	}

	return strings.Join(acneTypes[:len(acneTypes)-1], ", ") + " and " + acneTypes[len(acneTypes)-1]
}

func populateTreatmentPlan(dataApi api.DataAPI, patientVisitId int64, treatmentPlanId int64) (*common.TreatmentPlan, error) {
	treatmentPlan := &common.TreatmentPlan{
		PatientVisitId: encoding.NewObjectId(patientVisitId),
	}

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

func treatmentPlanResponse(dataApi api.DataAPI, w http.ResponseWriter, r *http.Request, treatmentPlan *common.TreatmentPlan, patientVisit *common.PatientVisit, doctor *common.Doctor, patient *common.Patient) {
	views := make([]TPView, 0)
	views = append(views, &TPVisitHeaderView{
		ImageURL: doctor.LargeThumbnailUrl,
		Title:    fmt.Sprintf("Dr. %s %s", doctor.FirstName, doctor.LastName),
		Subtitle: "Dermatologist",
	})

	views = append(views, &TPTextView{
		Text: treatmentPlan.DiagnosisSummary.Summary,
	})

	views = append(views, &TPImageView{
		ImageWidth:  125,
		ImageHeight: 45,
		ImageURL:    app_url.Asset(app_url.TmpSignature).String(),
	})

	views = append(views, &TPLargeDividerView{})

	if len(treatmentPlan.TreatmentList.Treatments) > 0 {
		views = append(views, &TPTextView{
			Text:  "Prescriptions",
			Style: "section_header",
		})

		for _, treatment := range treatmentPlan.TreatmentList.Treatments {
			views = append(views, &TPSmallDividerView{})

			iconURL := app_url.Asset(app_url.IconRX)
			if treatment.OTC {
				iconURL = app_url.Asset(app_url.IconOTC)
			}

			// only include tapurl and buttontitle if drug details
			// exist
			var buttonTitle string
			var tapUrl *app_url.SpruceAction
			if ndc := treatment.DrugDBIds[erx.NDC]; ndc != "" {
				if exists, err := dataApi.DoesDrugDetailsExist(ndc); exists {
					buttonTitle = "What to know about " + treatment.DrugName
					params := url.Values{}
					params.Set("treatment_id", strconv.FormatInt(treatment.Id.Int64(), 10))
					tapUrl = app_url.Action(app_url.ViewTreatmentGuideAction, params)
				} else if err != nil && err != api.NoRowsError {
					golog.Errorf("Error when trying to check if drug details exist: %s", err)
				}
			}

			views = append(views, &TPPrescriptionView{
				IconURL:     iconURL,
				Title:       fmt.Sprintf("%s %s", treatment.DrugInternalName, treatment.DosageStrength),
				Description: treatment.PatientInstructions,
				ButtonTitle: buttonTitle,
				TapURL:      tapUrl,
			})
		}
	}

	if treatmentPlan.RegimenPlan != nil && len(treatmentPlan.RegimenPlan.RegimenSections) > 0 {
		views = append(views, &TPLargeDividerView{})
		views = append(views, &TPTextView{
			Text:  "Personal Regimen",
			Style: sectionHeaderStyle,
		})

		for _, regimenSection := range treatmentPlan.RegimenPlan.RegimenSections {
			views = append(views, &TPSmallDividerView{})
			views = append(views, &TPTextView{
				Text:  regimenSection.RegimenName,
				Style: subheaderStyle,
			})

			for i, regimenStep := range regimenSection.RegimenSteps {
				views = append(views, &TPListElementView{
					ElementStyle: "numbered",
					Number:       i + 1,
					Text:         regimenStep.Text,
				})
			}
		}
	}

	if treatmentPlan.Advice != nil && len(treatmentPlan.Advice.SelectedAdvicePoints) > 0 {
		views = append(views, &TPLargeDividerView{})
		views = append(views, &TPTextView{
			Text:  fmt.Sprintf("Dr. %s's Advice", doctor.LastName),
			Style: sectionHeaderStyle,
		})

		switch len(treatmentPlan.Advice.SelectedAdvicePoints) {
		case 1:
			views = append(views, &TPTextView{
				Text: treatmentPlan.Advice.SelectedAdvicePoints[0].Text,
			})
		default:
			for _, advicePoint := range treatmentPlan.Advice.SelectedAdvicePoints {
				views = append(views, &TPListElementView{
					ElementStyle: "buletted",
					Text:         advicePoint.Text,
				})
			}
		}
	}

	views = append(views, &TPLargeDividerView{})
	views = append(views, &TPTextView{
		Text:  "Next Steps",
		Style: sectionHeaderStyle,
	})

	views = append(views, &TPSmallDividerView{})
	views = append(views, &TPTextView{
		Text: "Your prescriptions have been sent to your pharmacy and will be ready for pick soon.",
	})

	if patient.Pharmacy != nil {
		views = append(views, &TPSmallDividerView{})
		views = append(views, &TPTextView{
			Text:  "Your Pharmacy",
			Style: subheaderStyle,
		})

		views = append(views, &TPPharmacyMapView{
			Pharmacy: patient.Pharmacy,
		})
	}

	// identify prescriptions to pickup
	rxTreatments := make([]*common.Treatment, 0, len(treatmentPlan.TreatmentList.Treatments))
	for _, treatment := range treatmentPlan.TreatmentList.Treatments {
		if !treatment.OTC {
			rxTreatments = append(rxTreatments, treatment)
		}
	}

	if len(rxTreatments) > 0 {
		views = append(views, &TPSmallDividerView{})
		views = append(views, &TPTextView{
			Text:  "Prescriptions to Pick Up",
			Style: subheaderStyle,
		})

		treatmentListView := &TPTreatmentListView{}
		treatmentListView.Treatments = make([]*TPIconTextView, len(rxTreatments))
		for i, rxTreatment := range rxTreatments {
			treatmentListView.Treatments[i] = &TPIconTextView{
				IconURL:   app_url.Asset(app_url.IconRX),
				Text:      fmt.Sprintf("%s %s", rxTreatment.DrugInternalName, rxTreatment.DosageStrength),
				TextStyle: "bold",
			}
		}
		views = append(views, treatmentListView)
	}

	views = append(views, &TPButtonFooterView{
		FooterText: fmt.Sprintf("If you have any questions or concerns regarding your treatment plan, send Dr. %s a message.", doctor.LastName),
		ButtonText: fmt.Sprintf("Message Dr. %s", doctor.LastName),
		IconURL:    app_url.Asset(app_url.IconMessage),
		TapURL:     app_url.Action(app_url.MessageAction, nil),
	})

	for _, v := range views {
		if err := v.Validate(); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to render views: "+err.Error())
			return
		}
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, map[string][]TPView{"views": views})
}

package doctor_treatment_plan

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/golog"
	"errors"
	"fmt"
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

func fillInTreatmentPlan(drTreatmentPlan *common.DoctorTreatmentPlan, doctorId int64, dataApi api.DataAPI) error {
	var err error

	drTreatmentPlan.TreatmentList = &common.TreatmentList{}
	drTreatmentPlan.TreatmentList.Treatments, err = dataApi.GetTreatmentsBasedOnTreatmentPlanId(drTreatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get treatments for treatment plan: %s", err)
	}

	drTreatmentPlan.RegimenPlan = &common.RegimenPlan{}
	drTreatmentPlan.RegimenPlan, err = dataApi.GetRegimenPlanForTreatmentPlan(drTreatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get regimen plan for treatment plan: %s", err)
	}

	drTreatmentPlan.Advice = &common.Advice{}
	drTreatmentPlan.Advice.SelectedAdvicePoints, err = dataApi.GetAdvicePointsForTreatmentPlan(drTreatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get advice points for treatment plan")
	}

	// only populate the draft state if we are dealing with a draft treatment plan and the same doctor
	// that owns it is requesting the treatment plan (so that they can edit it)
	if drTreatmentPlan.DoctorId.Int64() == doctorId && drTreatmentPlan.Status == api.STATUS_DRAFT {
		drTreatmentPlan.RegimenPlan.AllRegimenSteps, err = dataApi.GetRegimenStepsForDoctor(drTreatmentPlan.DoctorId.Int64())
		if err != nil {
			return err
		}

		drTreatmentPlan.Advice.AllAdvicePoints, err = dataApi.GetAdvicePointsForDoctor(drTreatmentPlan.DoctorId.Int64())
		if err != nil {
			return err
		}

		setCommittedStateForEachSection(drTreatmentPlan)

		if err := populateContentSourceIntoTreatmentPlan(drTreatmentPlan, dataApi, doctorId); err == api.NoRowsError {
			return errors.New("No treatment plan found")
		} else if err != nil {
			return err
		}
	}
	return err
}

func setCommittedStateForEachSection(drTreatmentPlan *common.DoctorTreatmentPlan) {
	// depending on which sections have data in them, mark them to be committed or uncommitted
	// note that we intentionally treat a section with no data to be in the UNCOMMITTED state so as
	// to ensure that the doctor actually wanted to leave a particular section blank

	if len(drTreatmentPlan.TreatmentList.Treatments) > 0 {
		drTreatmentPlan.TreatmentList.Status = api.STATUS_COMMITTED
	} else {
		drTreatmentPlan.TreatmentList.Status = api.STATUS_UNCOMMITTED
	}

	if len(drTreatmentPlan.RegimenPlan.RegimenSections) > 0 {
		drTreatmentPlan.RegimenPlan.Status = api.STATUS_COMMITTED
	} else {
		drTreatmentPlan.RegimenPlan.Status = api.STATUS_UNCOMMITTED
	}

	if len(drTreatmentPlan.Advice.SelectedAdvicePoints) > 0 {
		drTreatmentPlan.Advice.Status = api.STATUS_COMMITTED
	} else {
		drTreatmentPlan.Advice.Status = api.STATUS_UNCOMMITTED
	}

}

func populateContentSourceIntoTreatmentPlan(treatmentPlan *common.DoctorTreatmentPlan, dataApi api.DataAPI, doctorId int64) error {
	// only continue if the content source of the treaetment plan is a favorite treatment plan
	if treatmentPlan.ContentSource == nil {
		return nil
	}

	switch treatmentPlan.ContentSource.ContentSourceType {
	case common.TPContentSourceTypeTreatmentPlan:
		previousTreatmentPlan, err := dataApi.GetTreatmentPlan(treatmentPlan.ContentSource.ContentSourceId.Int64(), doctorId)
		if err != nil {
			return err
		}

		if len(treatmentPlan.TreatmentList.Treatments) == 0 {
			fillTreatmentsIntoTreatmentPlan(previousTreatmentPlan.TreatmentList.Treatments, treatmentPlan)
		}

		if len(treatmentPlan.RegimenPlan.RegimenSections) == 0 {
			fillRegimenSectionsIntoTreatmentPlan(previousTreatmentPlan.RegimenPlan.RegimenSections, treatmentPlan)
		}

		if len(treatmentPlan.Advice.SelectedAdvicePoints) == 0 {
			fillAdvicePointsIntoTreatmentPlan(previousTreatmentPlan.Advice.SelectedAdvicePoints, treatmentPlan)
		}

	case common.TPContentSourceTypeFTP:
		favoriteTreatmentPlanId := treatmentPlan.ContentSource.ContentSourceId.Int64()

		favoriteTreatmentPlan, err := dataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId)
		if err != nil {
			return err
		}

		// The assumption here is that all components of a treatment plan that are already populated
		// match the items in the favorite treatment plan, if there exists a mapping to indicate that this
		// treatment plan must be filled in from a favorite treatment plan. The reason that we don't just write over
		// the items that do already belong in the treatment plan is to maintain the ids of the items that have been committed
		// to the database as part of the treatment plan.

		// populate treatments
		if len(treatmentPlan.TreatmentList.Treatments) == 0 {
			fillTreatmentsIntoTreatmentPlan(favoriteTreatmentPlan.TreatmentList.Treatments, treatmentPlan)
		}

		// populate regimen plan
		if len(treatmentPlan.RegimenPlan.RegimenSections) == 0 {
			fillRegimenSectionsIntoTreatmentPlan(favoriteTreatmentPlan.RegimenPlan.RegimenSections, treatmentPlan)
		}

		// populate advice
		if len(treatmentPlan.Advice.SelectedAdvicePoints) == 0 {
			fillAdvicePointsIntoTreatmentPlan(favoriteTreatmentPlan.Advice.SelectedAdvicePoints, treatmentPlan)
		}
	}

	return nil

}

func fillAdvicePointsIntoTreatmentPlan(sourceAdvicePoints []*common.DoctorInstructionItem, treatmentPlan *common.DoctorTreatmentPlan) {
	treatmentPlan.Advice.SelectedAdvicePoints = make([]*common.DoctorInstructionItem, len(sourceAdvicePoints))
	for i, advicePoint := range sourceAdvicePoints {
		treatmentPlan.Advice.SelectedAdvicePoints[i] = &common.DoctorInstructionItem{
			ParentId: advicePoint.ParentId,
			Text:     advicePoint.Text,
		}
	}
}

func fillRegimenSectionsIntoTreatmentPlan(sourceRegimenSections []*common.RegimenSection, treatmentPlan *common.DoctorTreatmentPlan) {
	treatmentPlan.RegimenPlan.RegimenSections = make([]*common.RegimenSection, len(sourceRegimenSections))

	for i, regimenSection := range sourceRegimenSections {
		treatmentPlan.RegimenPlan.RegimenSections[i] = &common.RegimenSection{
			RegimenName:  regimenSection.RegimenName,
			RegimenSteps: make([]*common.DoctorInstructionItem, len(regimenSection.RegimenSteps)),
		}

		for j, regimenStep := range regimenSection.RegimenSteps {
			treatmentPlan.RegimenPlan.RegimenSections[i].RegimenSteps[j] = &common.DoctorInstructionItem{
				ParentId: regimenStep.ParentId,
				Text:     regimenStep.Text,
			}
		}
	}
}

func fillTreatmentsIntoTreatmentPlan(sourceTreatments []*common.Treatment, treatmentPlan *common.DoctorTreatmentPlan) {
	treatmentPlan.TreatmentList.Treatments = make([]*common.Treatment, len(sourceTreatments))
	for i, treatment := range sourceTreatments {
		treatmentPlan.TreatmentList.Treatments[i] = &common.Treatment{
			DrugDBIds:               treatment.DrugDBIds,
			DrugInternalName:        treatment.DrugInternalName,
			DrugName:                treatment.DrugName,
			DrugRoute:               treatment.DrugRoute,
			DosageStrength:          treatment.DosageStrength,
			DispenseValue:           treatment.DispenseValue,
			DispenseUnitId:          treatment.DispenseUnitId,
			DispenseUnitDescription: treatment.DispenseUnitDescription,
			NumberRefills:           treatment.NumberRefills,
			SubstitutionsAllowed:    treatment.SubstitutionsAllowed,
			DaysSupply:              treatment.DaysSupply,
			PharmacyNotes:           treatment.PharmacyNotes,
			PatientInstructions:     treatment.PatientInstructions,
			CreationDate:            treatment.CreationDate,
			OTC:                     treatment.OTC,
			IsControlledSubstance:    treatment.IsControlledSubstance,
			SupplementalInstructions: treatment.SupplementalInstructions,
		}
	}
}

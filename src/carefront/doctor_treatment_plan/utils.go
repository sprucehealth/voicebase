package doctor_treatment_plan

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/golog"
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

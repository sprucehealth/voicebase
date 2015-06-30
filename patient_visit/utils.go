package patient_visit

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
)

const (
	acneDiagnosisQuestionTag         = "q_acne_diagnosis"
	acneTypeQuestionTag              = "q_acne_type"
	rosaceaTypeQuestionTag           = "q_acne_rosacea_type"
	acneDescribeConditionQuestionTag = "q_diagnosis_describe_condition"
	notSuitableReasonQuestionTag     = "q_diagnosis_reason_not_suitable"

	acneVulgarisAnswerTag           = "a_doctor_acne_vulgaris"
	acneRosaceaAnswerTag            = "a_doctor_acne_rosacea"
	acnePerioralDermatitisAnswerTag = "a_doctor_acne_perioral_dermatitis"
	acneSomethingElseAnswerTag      = "a_doctor_acne_something_else"
	notSuitableForSpruceAnswerTag   = "a_doctor_acne_not_suitable_spruce"

	acneTypeComedonalAnswerTag    = "a_acne_comedonal"
	acneTypeInflammatoryAnswerTag = "a_acne_inflammatory"
	acneTypeCysticAnswerTag       = "a_acne_cysts"
	acneTypeHormonalAnswerTag     = "a_acne_hormonal"

	acneTypeErythematotelangiectaticAnswerTag = "a_acne_erythematotelangiectatic_rosacea"
	acneTypePapulopstularAnswerTag            = "a_acne_papulopstular_rosacea"
	acneTypeRhinophymaAnswerTag               = "a_acne_rhinophyma_rosacea"
	acneTypeOcularAnswerTag                   = "a_acne_ocular_rosacea"
)

var (
	cachedQuestionIds = make(map[string]int64)
	cachedAnswerIds   = make(map[int64]*info_intake.PotentialAnswer)
	cacheOnce         sync.Once
)

func cacheInfoForUnsuitableVisit(dataAPI api.DataAPI) {
	cacheOnce.Do(func() {
		// cache question ids
		questionInfoList, err := dataAPI.GetQuestionInfoForTags([]string{acneDiagnosisQuestionTag, acneTypeQuestionTag, rosaceaTypeQuestionTag, acneDescribeConditionQuestionTag, notSuitableReasonQuestionTag}, api.LanguageIDEnglish)
		if err != nil {
			panic(err)
		} else {
			for _, qInfo := range questionInfoList {
				cachedQuestionIds[qInfo.QuestionTag] = qInfo.QuestionID
			}
		}

		// cache answerS
		answerInfoList, err := dataAPI.GetAnswerInfoForTags([]string{acneVulgarisAnswerTag, acneRosaceaAnswerTag, acnePerioralDermatitisAnswerTag, acneSomethingElseAnswerTag, notSuitableForSpruceAnswerTag,
			acneTypeComedonalAnswerTag, acneTypeInflammatoryAnswerTag, acneTypeCysticAnswerTag, acneTypeHormonalAnswerTag,
			acneTypeErythematotelangiectaticAnswerTag, acneTypePapulopstularAnswerTag, acneTypeRhinophymaAnswerTag, acneTypeOcularAnswerTag}, api.LanguageIDEnglish)
		if err != nil {
			panic(err)
		} else {
			for _, aInfo := range answerInfoList {
				cachedAnswerIds[aInfo.AnswerID] = aInfo
			}
		}
	})
}

func GetDiagnosisLayout(dataAPI api.DataAPI, patientVisit *common.PatientVisit, doctorID int64) (*info_intake.DiagnosisIntake, error) {
	// TODO: assume Acne
	pathway, err := dataAPI.PathwayForTag(api.AcnePathwayTag, api.PONone)
	if err != nil {
		return nil, err
	}

	diagnosisLayout, err := getCurrentActiveDiagnoseLayoutForHealthCondition(dataAPI, pathway.ID)
	if err != nil {
		return nil, err
	}
	diagnosisLayout.PatientVisitID = patientVisit.ID.Int64()

	// get a list of question ids in ther diagnosis layout, so that we can look for answers from the doctor pertaining to this visit
	questionIDs := getQuestionIdsInDiagnosisLayout(diagnosisLayout)

	// get the answers to the questions in the array
	doctorAnswers, err := dataAPI.AnswersForQuestions(questionIDs, &api.DiagnosisIntake{
		DoctorID:       doctorID,
		PatientVisitID: patientVisit.ID.Int64(),
	})
	if err != nil {
		return nil, err
	}

	// if the doctor is dealing with a followup and the doctor's diagnosis does not
	// exist for the followup yet, prepopulate the diagnosis with the previous treated visit's
	// information
	if patientVisit.IsFollowup && len(doctorAnswers) == 0 {
		visits, err := dataAPI.GetVisitsForCase(patientVisit.PatientCaseID.Int64(), common.TreatedPatientVisitStates())
		if err != nil {
			return nil, err
		}

		doctorAnswers, err = dataAPI.AnswersForQuestions(questionIDs, &api.DiagnosisIntake{
			DoctorID:       doctorID,
			PatientVisitID: visits[0].ID.Int64(),
		})
		if err != nil {
			return nil, err
		}
	}

	// populate the diagnosis layout with the answers to the questions
	populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout, doctorAnswers)
	return diagnosisLayout, nil
}

func wasVisitMarkedUnsuitableForSpruce(intakeData *apiservice.IntakeData) (string, bool) {
	var reasonMarkedUnsuitable string
	var wasMarkedUnsuitable bool
	for _, questionItem := range intakeData.Questions {
		if questionItem.QuestionID == cachedQuestionIds[acneDiagnosisQuestionTag] {
			if cachedAnswerIds[questionItem.AnswerIntakes[0].PotentialAnswerID].AnswerTag == notSuitableForSpruceAnswerTag {
				wasMarkedUnsuitable = true
			}
		} else if questionItem.QuestionID == cachedQuestionIds[notSuitableReasonQuestionTag] {
			reasonMarkedUnsuitable = questionItem.AnswerIntakes[0].AnswerText
		}
	}
	return reasonMarkedUnsuitable, wasMarkedUnsuitable
}

func determineDiagnosisFromAnswers(intakeData *apiservice.IntakeData) string {
	// first identify the types of acne, if picked
	var diagnosisType string
	for _, questionItem := range intakeData.Questions {
		if questionItem.QuestionID == cachedQuestionIds[acneTypeQuestionTag] || questionItem.QuestionID == cachedQuestionIds[rosaceaTypeQuestionTag] {

			var dTypes []string
			for _, answerItem := range questionItem.AnswerIntakes {
				dTypes = append(dTypes, cachedAnswerIds[answerItem.PotentialAnswerID].Answer)
			}
			if len(dTypes) == 1 {
				diagnosisType = dTypes[0]
			} else {
				diagnosisType = strings.Join(dTypes[:len(dTypes)-1], ", ") + " and " + dTypes[len(dTypes)-1]
			}
		}
	}

	for _, questionItem := range intakeData.Questions {

		switch questionItem.QuestionID {

		// if the doctor answered the question to describe the condition, then
		// the entered description is picked as the diagnosis because the doctor is only
		// prompted to describe the condition if the doctor picks "Something else" as the diagnosis category
		case cachedQuestionIds[acneDescribeConditionQuestionTag]:
			return questionItem.AnswerIntakes[0].AnswerText

		// if the doctor picked one of the other diagnosis, then we combined
		// the overarching diagnosis with the type of diagnosis
		case cachedQuestionIds[acneDiagnosisQuestionTag]:
			diagnosisCategoryAnswerInfo := cachedAnswerIds[questionItem.AnswerIntakes[0].PotentialAnswerID]
			switch diagnosisCategoryAnswerInfo.AnswerTag {

			case acnePerioralDermatitisAnswerTag:
				return diagnosisCategoryAnswerInfo.Answer

			case acneVulgarisAnswerTag:
				return diagnosisType + " Acne"

			case acneRosaceaAnswerTag:
				return diagnosisType + " Rosacea"
			}
		}
	}

	return ""
}

func getQuestionIdsInDiagnosisLayout(diagnosisLayout *info_intake.DiagnosisIntake) []int64 {
	questionIDs := make([]int64, 0)
	for _, section := range diagnosisLayout.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			questionIDs = append(questionIDs, question.QuestionID)
		}
	}

	return questionIDs
}

func populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout *info_intake.DiagnosisIntake, doctorAnswers map[int64][]common.Answer) []int64 {
	questionIDs := make([]int64, 0)
	for _, section := range diagnosisLayout.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			// go through each question to see if there exists a patient answer for it
			question.Answers = doctorAnswers[question.QuestionID]
		}
	}

	return questionIDs
}

func getCurrentActiveDiagnoseLayoutForHealthCondition(dataAPI api.DataAPI, pathwayID int64) (*info_intake.DiagnosisIntake, error) {
	layoutVersion, err := dataAPI.GetActiveDoctorDiagnosisLayout(pathwayID)
	if err != nil {
		return nil, err
	}

	var diagnosisLayout info_intake.DiagnosisIntake
	if err = json.Unmarshal(layoutVersion.Layout, &diagnosisLayout); err != nil {
		return nil, err
	}

	return &diagnosisLayout, nil
}

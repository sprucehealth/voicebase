package patient_visit

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
)

const (
	acneDiagnosisQuestionTag         = "q_acne_diagnosis"
	acneTypeQuestionTag              = "q_acne_type"
	acneDescribeConditionQuestionTag = "q_diagnosis_describe_condition"

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

var notSuitableForSpruceAnswerId int64
var acneDiagnosisQuestionId int64

var cachedQuestionIds = make(map[string]int64)
var cachedAnswerIds = make(map[int64]*info_intake.PotentialAnswer)

func cacheInfoForUnsuitableVisit(dataApi api.DataAPI) {
	// cache question ids
	questionInfoList, err := dataApi.GetQuestionInfoForTags([]string{acneDiagnosisQuestionTag, acneTypeQuestionTag, acneDescribeConditionQuestionTag}, api.EN_LANGUAGE_ID)
	if err != nil {
		panic(err)
	} else {
		for _, qInfo := range questionInfoList {
			cachedQuestionIds[qInfo.QuestionTag] = qInfo.QuestionId
		}
	}

	// cache answerS
	answerInfoList, err := dataApi.GetAnswerInfoForTags([]string{acneVulgarisAnswerTag, acneRosaceaAnswerTag, acnePerioralDermatitisAnswerTag, acneSomethingElseAnswerTag, notSuitableForSpruceAnswerTag,
		acneTypeComedonalAnswerTag, acneTypeInflammatoryAnswerTag, acneTypeCysticAnswerTag, acneTypeHormonalAnswerTag,
		acneTypeErythematotelangiectaticAnswerTag, acneTypePapulopstularAnswerTag, acneTypeRhinophymaAnswerTag, acneTypeOcularAnswerTag}, api.EN_LANGUAGE_ID)
	if err != nil {
		panic(err)
	} else {
		for _, aInfo := range answerInfoList {
			cachedAnswerIds[aInfo.AnswerId] = aInfo
		}
	}
}

func GetDiagnosisLayout(dataApi api.DataAPI, patientVisitId, doctorId int64) (*info_intake.DiagnosisIntake, error) {

	diagnosisLayout, err := getCurrentActiveDiagnoseLayoutForHealthCondition(dataApi, apiservice.HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		return nil, err
	}
	diagnosisLayout.PatientVisitId = patientVisitId

	// get a list of question ids in ther diagnosis layout, so that we can look for answers from the doctor pertaining to this visit
	questionIds := getQuestionIdsInDiagnosisLayout(diagnosisLayout)

	// get the answers to the questions in the array
	doctorAnswers, err := dataApi.GetDoctorAnswersForQuestionsInDiagnosisLayout(questionIds, doctorId, patientVisitId)
	if err != nil {
		return nil, err
	}

	// populate the diagnosis layout with the answers to the questions
	populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout, doctorAnswers)
	return diagnosisLayout, nil
}

func populateGlobalSectionsWithPatientAnswers(dataApi api.DataAPI, healthCondition *info_intake.InfoIntakeLayout, patientId int64, r *http.Request) error {
	// identify sections that are global
	globalSectionIds, err := dataApi.GetGlobalSectionIds()
	if err != nil {
		return errors.New("Unable to get global sections ids: " + err.Error())
	}

	globalQuestionIds := make([]int64, 0)
	for _, sectionId := range globalSectionIds {
		questionIds := getQuestionIdsInSectionInIntakeLayout(healthCondition, sectionId)
		globalQuestionIds = append(globalQuestionIds, questionIds...)
	}

	// get the answers that the patient has previously entered for all sections that are considered global
	globalSectionPatientAnswers, err := dataApi.GetPatientAnswersForQuestionsInGlobalSections(globalQuestionIds, patientId)
	if err != nil {
		return errors.New("Unable to get patient answers for global sections: " + err.Error())
	}

	populateIntakeLayoutWithPatientAnswers(healthCondition, globalSectionPatientAnswers, r)
	return nil
}

func populateSectionsWithPatientAnswers(dataApi api.DataAPI, patientId, patientVisitId int64, patientVisitLayout *info_intake.InfoIntakeLayout, r *http.Request) error {
	// get answers that the patient has previously entered for this particular patient visit
	// and feed the answers into the layout
	questionIdsInAllSections := apiservice.GetNonPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)
	photoQuestionIds := apiservice.GetPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)

	patientAnswersForVisit, err := dataApi.GetPatientAnswersForQuestionsBasedOnQuestionIds(questionIdsInAllSections, patientId, patientVisitId)
	if err != nil {
		return err
	}

	photoSectionsByQuestion, err := dataApi.GetPatientCreatedPhotoSectionsForQuestionIds(photoQuestionIds, patientId, patientVisitId)
	if err != nil {
		return err
	}

	for questionId, answers := range photoSectionsByQuestion {
		patientAnswersForVisit[questionId] = answers
	}

	populateIntakeLayoutWithPatientAnswers(patientVisitLayout, patientAnswersForVisit, r)
	return nil
}

func getQuestionIdsInSectionInIntakeLayout(healthCondition *info_intake.InfoIntakeLayout, sectionId int64) (questionIds []int64) {
	questionIds = make([]int64, 0)
	for _, section := range healthCondition.Sections {
		if section.SectionId == sectionId {
			for _, screen := range section.Screens {
				for _, question := range screen.Questions {
					questionIds = append(questionIds, question.QuestionId)
				}
			}
		}
	}
	return
}

func wasVisitMarkedUnsuitableForSpruce(answerIntakeRequestBody *apiservice.AnswerIntakeRequestBody) bool {
	for _, questionItem := range answerIntakeRequestBody.Questions {
		if questionItem.QuestionId == cachedQuestionIds[acneDiagnosisQuestionTag] {
			if cachedAnswerIds[questionItem.AnswerIntakes[0].PotentialAnswerId].AnswerTag == notSuitableForSpruceAnswerTag {
				return true
			}
		}
	}
	return false
}

func determineDiagnosisFromAnswers(answerIntakeRequestBody *apiservice.AnswerIntakeRequestBody) string {
	// first identify the type of acne, if one was picked
	var diagnosisType string
	for _, questionItem := range answerIntakeRequestBody.Questions {
		if questionItem.QuestionId == cachedQuestionIds[acneTypeQuestionTag] {
			diagnosisType = cachedAnswerIds[questionItem.AnswerIntakes[0].PotentialAnswerId].Answer
		}
	}

	for _, questionItem := range answerIntakeRequestBody.Questions {

		switch questionItem.QuestionId {

		// if the doctor answered the question to describe the condition, then
		// the entered description is picked as the diagnosis because the doctor is only
		// prompted to describe the condition if the doctor picks "Something else" as the diagnosis category
		case cachedQuestionIds[acneDescribeConditionQuestionTag]:
			return questionItem.AnswerIntakes[0].AnswerText

		// if the doctor picked one of the other diagnosis, then we combined
		// the overarching diagnosis with the type of diagnosis
		case cachedQuestionIds[acneDiagnosisQuestionTag]:
			diagnosisCategoryAnswerInfo := cachedAnswerIds[questionItem.AnswerIntakes[0].PotentialAnswerId]
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
	questionIds := make([]int64, 0)
	for _, section := range diagnosisLayout.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			questionIds = append(questionIds, question.QuestionId)
		}
	}

	return questionIds
}

func populateDiagnosisLayoutWithDoctorAnswers(diagnosisLayout *info_intake.DiagnosisIntake, doctorAnswers map[int64][]common.Answer) []int64 {
	questionIds := make([]int64, 0)
	for _, section := range diagnosisLayout.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {
			// go through each question to see if there exists a patient answer for it
			question.Answers = doctorAnswers[question.QuestionId]
		}
	}

	return questionIds
}

func getCurrentActiveDiagnoseLayoutForHealthCondition(dataApi api.DataAPI, healthConditionId int64) (*info_intake.DiagnosisIntake, error) {
	data, _, err := dataApi.GetActiveDoctorDiagnosisLayout(healthConditionId)
	if err != nil {
		return nil, err
	}

	var diagnosisLayout info_intake.DiagnosisIntake
	if err = json.Unmarshal(data, &diagnosisLayout); err != nil {
		return nil, err
	}

	return &diagnosisLayout, nil
}

func populateIntakeLayoutWithPatientAnswers(intake *info_intake.InfoIntakeLayout, patientAnswers map[int64][]common.Answer, r *http.Request) {
	for _, section := range intake.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				// go through each question to see if there exists a patient answer for it
				question.Answers = patientAnswers[question.QuestionId]
				if question.QuestionType == info_intake.QUESTION_TYPE_PHOTO_SECTION {
					if len(question.Answers) > 0 {
						// go through each slot and populate the url for the photo
						for _, answer := range question.Answers {
							photoIntakeSection := answer.(*common.PhotoIntakeSection)
							for _, photoIntakeSlot := range photoIntakeSection.Photos {
								photoIntakeSlot.PhotoUrl = apiservice.CreatePhotoUrl(photoIntakeSlot.PhotoId, photoIntakeSection.Id, common.ClaimerTypePhotoIntakeSection, r.Host)
							}
						}
					}

				}
			}
		}
	}
}

func getCurrentActiveClientLayoutForHealthCondition(dataApi api.DataAPI, healthConditionId, languageId int64) (*info_intake.InfoIntakeLayout, int64, error) {
	data, layoutVersionId, err := dataApi.GetCurrentActivePatientLayout(languageId, healthConditionId)
	if err != nil {
		return nil, 0, err
	}

	patientVisitLayout := &info_intake.InfoIntakeLayout{}
	if err := json.Unmarshal(data, patientVisitLayout); err != nil {
		return nil, 0, err
	}
	return patientVisitLayout, layoutVersionId, nil
}

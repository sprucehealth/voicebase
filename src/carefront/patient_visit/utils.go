package patient_visit

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/info_intake"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func fillInFormattedFieldsForQuestions(healthCondition *info_intake.InfoIntakeLayout, doctor *common.Doctor) {
	for _, section := range healthCondition.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {

				if question.FormattedFieldTags != nil {

					// populate the values for each of the fields in order
					for _, fieldTag := range question.FormattedFieldTags {
						fieldTagComponents := strings.Split(fieldTag, ":")
						if fieldTagComponents[0] == info_intake.FORMATTED_TITLE_FIELD {
							switch fieldTagComponents[1] {
							case info_intake.FORMATTED_FIELD_DOCTOR_LAST_NAME:
								// build the formatted string and assign it back to the question title
								question.QuestionTitle = fmt.Sprintf(question.QuestionTitle, strings.Title(doctor.LastName))
							}
						}
					}

				}
			}
		}
	}
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
								photoIntakeSlot.PhotoUrl = apiservice.CreatePhotoUrl(photoIntakeSlot.PhotoId, photoIntakeSlot.Id, common.ClaimerTypePhotoIntakeSlot, r.Host)
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

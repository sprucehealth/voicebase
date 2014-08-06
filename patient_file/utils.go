package patient_file

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
)

// This interface is used to populate the ViewContext with data pertaining to a single question
type patientQAViewContextPopulator interface {
	populateViewContextWithPatientQA(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, apiDomain string) error
}

var patientQAPopulators map[string]patientQAViewContextPopulator = make(map[string]patientQAViewContextPopulator, 0)

func init() {
	patientQAPopulators[info_intake.QUESTION_TYPE_AUTOCOMPLETE] = qaViewContextPopulator(populateAnswersForQuestionsWithSubanswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_MULTIPLE_CHOICE] = qaViewContextPopulator(populateMultipleChoiceAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_SINGLE_ENTRY] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_FREE_TEXT] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_SINGLE_SELECT] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_PHOTO_SECTION] = qaViewContextPopulator(populatePatientPhotos)
	patientQAPopulators[info_intake.QUESTION_TYPE_PHOTO] = qaViewContextPopulator(populatePatientPhotos)
}

type qaViewContextPopulator func([]common.Answer, *info_intake.Question, *common.ViewContext, api.DataAPI, string) error

func (q qaViewContextPopulator) populateViewContextWithPatientQA(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, apiDomain string) error {
	return q(patientAnswers, question, context, dataApi, apiDomain)
}

func populateMultipleChoiceAnswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, apiDomain string) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	// if we are dealing with a question that has subquestions defined, populate the context appropriately
	if question.SubQuestionsConfig != nil && (len(question.SubQuestionsConfig.Screens) > 0 || len(question.SubQuestionsConfig.Questions) > 0) {
		return populateAnswersForQuestionsWithSubanswers(patientAnswers, question, context, dataApi, apiDomain)
	}

	checkedUncheckedItems := make([]info_intake.CheckedUncheckedData, len(question.PotentialAnswers))
	for i, potentialAnswer := range question.PotentialAnswers {
		answerSelected := false

		for _, patientAnswer := range patientAnswers {
			pAnswer := patientAnswer.(*common.AnswerIntake)
			if pAnswer.PotentialAnswerId.Int64() == potentialAnswer.AnswerId {
				answerSelected = true
			}
		}

		checkedUncheckedItems[i] = info_intake.CheckedUncheckedData{
			Value:     potentialAnswer.Answer,
			IsChecked: answerSelected,
		}
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), checkedUncheckedItems)
	return nil
}

func populateSingleEntryAnswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, apiDomain string) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	if len(patientAnswers) > 1 {
		return fmt.Errorf("Expected just one answer for question %s instead we have  %d", question.QuestionTag, len(patientAnswers))
	}

	pAnswer := patientAnswers[0].(*common.AnswerIntake)
	answer := pAnswer.AnswerText
	if answer == "" {
		answer = pAnswer.AnswerSummary
	}
	if answer == "" {
		answer = pAnswer.PotentialAnswer
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), answer)
	return nil
}

func populateAnswersForQuestionsWithSubanswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, apiDomain string) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	// creating a mapping of id to subquestion
	qMapping := make(map[int64]*info_intake.Question)
	if question.SubQuestionsConfig != nil {
		for _, subQuestion := range question.SubQuestionsConfig.Questions {
			qMapping[subQuestion.QuestionId] = subQuestion
		}

		for _, screen := range question.SubQuestionsConfig.Screens {
			for _, subQuestion := range screen.Questions {
				qMapping[subQuestion.QuestionId] = subQuestion
			}
		}
	}

	data := make([]info_intake.TitleSubItemsDescriptionContentData, len(patientAnswers))
	for i, patientAnswer := range patientAnswers {
		pAnswer := patientAnswer.(*common.AnswerIntake)
		items := make([]*info_intake.DescriptionContentData, 0, len(pAnswer.SubAnswers))
		for _, subAnswer := range pAnswer.SubAnswers {
			// user-entered answer gets priority, then any summary for an answer, followed by the potential answer
			// if it exists
			if subAnswer.AnswerText != "" {
				items = append(items, &info_intake.DescriptionContentData{
					Description: qMapping[subAnswer.QuestionId.Int64()].QuestionSummary,
					Content:     subAnswer.AnswerText,
				})
			} else if subAnswer.AnswerSummary != "" {
				items = append(items, &info_intake.DescriptionContentData{
					Description: qMapping[subAnswer.QuestionId.Int64()].QuestionSummary,
					Content:     subAnswer.AnswerSummary,
				})
			} else if subAnswer.PotentialAnswer != "" {
				items = append(items, &info_intake.DescriptionContentData{
					Description: qMapping[subAnswer.QuestionId.Int64()].QuestionSummary,
					Content:     subAnswer.PotentialAnswer,
				})
			}
		}

		// title is either a user-defined entry or the potential answer
		title := pAnswer.AnswerText
		if title == "" {
			title = pAnswer.PotentialAnswer
		}

		data[i] = info_intake.TitleSubItemsDescriptionContentData{
			Title:    title,
			SubItems: items,
		}
	}
	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), data)
	return nil
}

// if there are no patient answers for this question,
// check if the empty state text is specified in the additional fields
// of the question
func populateEmptyStateTextIfPresent(question *info_intake.Question, context *common.ViewContext) {
	emptyStateText, ok := question.AdditionalFields["empty_state_text"]
	if !ok {
		return
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:empty_state_text", question.QuestionTag), emptyStateText)
}

func populatePatientPhotos(answeredPhotoSections []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, apiDomain string) error {
	var items []info_intake.TitlePhotoListData
	photoData, ok := context.Get("patient_visit_photos")

	if !ok || photoData == nil {
		items = make([]info_intake.TitlePhotoListData, 0, len(answeredPhotoSections))
	} else {
		items = photoData.([]info_intake.TitlePhotoListData)
	}

	for _, photoSection := range answeredPhotoSections {
		pIntakeSection := photoSection.(*common.PhotoIntakeSection)
		item := info_intake.TitlePhotoListData{
			Title:  pIntakeSection.Name,
			Photos: make([]info_intake.PhotoData, len(pIntakeSection.Photos)),
		}

		for i, photoIntakeSlot := range pIntakeSection.Photos {
			item.Photos[i] = info_intake.PhotoData{
				Title:    photoIntakeSlot.Name,
				PhotoUrl: apiservice.CreatePhotoUrl(photoIntakeSlot.PhotoId, pIntakeSection.Id, common.ClaimerTypePhotoIntakeSection, apiDomain),
			}
		}
		items = append(items, item)
	}

	context.Set("patient_visit_photos", items)
	return nil
}

func buildContext(dataApi api.DataAPI, patientVisitLayout *info_intake.InfoIntakeLayout, patientId, patientVisitId int64, apiDomain string) (common.ViewContext, error) {
	questions := apiservice.GetQuestionsInPatientVisitLayout(patientVisitLayout)

	questionIds := apiservice.GetNonPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)
	photoQuestionIds := apiservice.GetPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)

	// get all the answers the patient entered for the questions (note that there may not be an answer for every question)
	patientAnswersForQuestions, err := dataApi.GetPatientAnswersForQuestions(questionIds, patientId, patientVisitId)
	if err != nil {
		return nil, err
	}

	photoSectionsByQuestion, err := dataApi.GetPatientCreatedPhotoSectionsForQuestionIds(photoQuestionIds, patientId, patientVisitId)
	if err != nil {
		return nil, err
	}

	// combine photo sections into the patient answers
	for questionId, photoSections := range photoSectionsByQuestion {
		patientAnswersForQuestions[questionId] = photoSections
	}

	context, err := populateContextForRenderingLayout(patientAnswersForQuestions, questions, dataApi, patientId, apiDomain)
	if err != nil {
		return nil, err
	}

	return context, err
}

func populateContextForRenderingLayout(patientAnswersForQuestions map[int64][]common.Answer, questions []*info_intake.Question, dataApi api.DataAPI, patientId int64, apiDomain string) (common.ViewContext, error) {
	context := common.NewViewContext()

	alerts, err := dataApi.GetAlertsForPatient(patientId)
	if err != nil {
		return nil, err
	} else if len(alerts) > 0 {
		alertsArray := make([]string, len(alerts))
		for i, alert := range alerts {
			alertsArray[i] = alert.Message
		}
		context.Set("patient_visit_alerts", alertsArray)
	} else {
		context.Set("patient_visit_alerts:empty_state_text", "No alerts")
	}

	// go through each question
	for _, question := range questions {
		contextPopulator, ok := patientQAPopulators[question.QuestionType]
		if !ok {
			return nil, fmt.Errorf("Context populator not found for question with type %s", question.QuestionType)
		}

		if err := contextPopulator.populateViewContextWithPatientQA(patientAnswersForQuestions[question.QuestionId], question, context, dataApi, apiDomain); err != nil {
			return nil, err
		}
	}

	return *context, nil
}

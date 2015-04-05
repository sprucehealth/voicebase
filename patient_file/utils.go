package patient_file

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
)

// This interface is used to populate the ViewContext with data pertaining to a single question
type patientQAViewContextPopulator interface {
	populateViewContextWithPatientQA(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error
}

var patientQAPopulators map[string]patientQAViewContextPopulator = make(map[string]patientQAViewContextPopulator, 0)

func init() {
	patientQAPopulators[info_intake.QuestionTypeAutocomplete] = qaViewContextPopulator(populateAnswersForQuestionsWithSubanswers)
	patientQAPopulators[info_intake.QuestionTypeMultipleChoice] = qaViewContextPopulator(populateMultipleChoiceAnswers)
	patientQAPopulators[info_intake.QuestionTypeSingleEntry] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QuestionTypeFreeText] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QuestionTypeSingleSelect] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QuestionTypeSegmentedControl] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QuestionTypePhotoSection] = qaViewContextPopulator(populatePatientPhotos)
	patientQAPopulators[info_intake.QuestionTypePhoto] = qaViewContextPopulator(populatePatientPhotos)
}

type qaViewContextPopulator func([]common.Answer, *info_intake.Question, *common.ViewContext) error

func (q qaViewContextPopulator) populateViewContextWithPatientQA(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
	return q(patientAnswers, question, context)
}

func populateMultipleChoiceAnswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	// if we are dealing with a question that has subquestions defined, populate the context appropriately
	if question.SubQuestionsConfig != nil && (len(question.SubQuestionsConfig.Screens) > 0 || len(question.SubQuestionsConfig.Questions) > 0) {
		return populateAnswersForQuestionsWithSubanswers(patientAnswers, question, context)
	}

	checkedUncheckedItems := make([]info_intake.CheckedUncheckedData, 0)
	var otherTextEntries []string
	for _, potentialAnswer := range question.PotentialAnswers {
		answerSelected := false
		answerText := potentialAnswer.Answer
		otherTextEntries = otherTextEntries[:0]
		for _, patientAnswer := range patientAnswers {
			pAnswer := patientAnswer.(*common.AnswerIntake)
			if pAnswer.PotentialAnswerID.Int64() == potentialAnswer.AnswerID {
				answerSelected = true
				if pAnswer.AnswerText != "" {
					otherTextEntries = append(otherTextEntries, pAnswer.AnswerText)
				}
			}
		}

		if len(otherTextEntries) > 0 {
			answerText = fmt.Sprintf("%s - %s", answerText, strings.Join(otherTextEntries, ", "))
		}

		checkedUncheckedItems = append(checkedUncheckedItems, info_intake.CheckedUncheckedData{
			Value:     answerText,
			IsChecked: answerSelected,
		})
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), checkedUncheckedItems)
	return nil
}

func populateSingleEntryAnswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
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

func populateAnswersForQuestionsWithSubanswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	// creating a mapping of id to subquestion
	qMapping := make(map[int64]*info_intake.Question)
	if question.SubQuestionsConfig != nil {
		for _, subQuestion := range question.SubQuestionsConfig.Questions {
			qMapping[subQuestion.QuestionID] = subQuestion
		}

		for _, screen := range question.SubQuestionsConfig.Screens {
			for _, subQuestion := range screen.Questions {
				qMapping[subQuestion.QuestionID] = subQuestion
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
					Description: qMapping[subAnswer.QuestionID.Int64()].QuestionSummary,
					Content:     subAnswer.AnswerText,
				})
			} else if subAnswer.AnswerSummary != "" {
				items = append(items, &info_intake.DescriptionContentData{
					Description: qMapping[subAnswer.QuestionID.Int64()].QuestionSummary,
					Content:     subAnswer.AnswerSummary,
				})
			} else if subAnswer.PotentialAnswer != "" {
				items = append(items, &info_intake.DescriptionContentData{
					Description: qMapping[subAnswer.QuestionID.Int64()].QuestionSummary,
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

func populatePatientPhotos(answeredPhotoSections []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
	if len(answeredPhotoSections) == 0 {
		return nil
	}

	var items []info_intake.TitlePhotoListData
	// continue to populate the global patient_visit_photos
	// key for backwards compatibility, given that acne related
	// doctor reviews expect this key to exist.
	photoData, ok := context.Get("patient_visit_photos")

	if !ok || photoData == nil {
		items = make([]info_intake.TitlePhotoListData, 0, len(answeredPhotoSections))
	} else {
		items = photoData.([]info_intake.TitlePhotoListData)
	}

	// keep track of the question specific items so that we can create a key to link to
	// photos pertaining to a question
	questionSpecificItems := make([]info_intake.TitlePhotoListData, 0, len(answeredPhotoSections))

	for _, photoSection := range answeredPhotoSections {
		pIntakeSection := photoSection.(*common.PhotoIntakeSection)
		item := info_intake.TitlePhotoListData{
			Title:  pIntakeSection.Name,
			Photos: make([]info_intake.PhotoData, len(pIntakeSection.Photos)),
		}

		for i, photoIntakeSlot := range pIntakeSection.Photos {
			item.Photos[i] = info_intake.PhotoData{
				Title:    photoIntakeSlot.Name,
				PhotoID:  photoIntakeSlot.PhotoID,
				PhotoURL: photoIntakeSlot.PhotoURL,
			}
		}
		items = append(items, item)
		questionSpecificItems = append(questionSpecificItems, item)
	}

	context.Set("patient_visit_photos", items)
	context.Set(question.QuestionTag+":photos", questionSpecificItems)
	return nil
}

func buildContext(
	dataAPI api.DataAPI,
	visitLayout *info_intake.InfoIntakeLayout,
	visit *common.PatientVisit) (*common.ViewContext, error) {

	context, err := populateContextForRenderingLayout(
		visitLayout.Answers(),
		visitLayout.Questions(),
		dataAPI,
		visit.PatientID.Int64(),
		visit.PatientVisitID.Int64())

	if err != nil {
		return nil, err
	}

	return context, err
}

func populateContextForRenderingLayout(
	answers map[int64][]common.Answer,
	questions []*info_intake.Question,
	dataAPI api.DataAPI, patientID, patientVisitID int64) (*common.ViewContext, error) {
	context := common.NewViewContext(nil)

	// populate alerts
	alerts, err := dataAPI.AlertsForVisit(patientVisitID)
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

	// populate message for patient visit if one exists
	message, err := dataAPI.GetMessageForPatientVisit(patientVisitID)
	if err != nil && !api.IsErrNotFound(err) {
		return nil, err
	}
	if message != "" {
		context.Set("q_anything_else_acne:answers", message)
		context.Set("visit_message", message)
	} else {
		context.Set("visit_message:empty_state_text", "Patient did not specify")
	}

	// go through each question
	for _, question := range questions {
		contextPopulator, ok := patientQAPopulators[question.QuestionType]
		if !ok {
			return nil, fmt.Errorf("Context populator not found for question with type %s", question.QuestionType)
		}

		if err := contextPopulator.populateViewContextWithPatientQA(answers[question.QuestionID], question, context); err != nil {
			return nil, err
		}
	}

	return context, nil
}

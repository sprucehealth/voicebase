package patient_file

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/info_intake"
	"fmt"
	"strings"
)

// This interface is used to populate the ViewContext with data pertaining to a single question
type patientQAViewContextPopulator interface {
	populateViewContextWithPatientQA(patientAnswers []*common.AnswerIntake, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, photoStorageService api.CloudStorageAPI) error
}

// This interface is used to populate the ViewContext with any global data or business logic
type genericPatientViewContextPopulator interface {
	populateViewContextWithInfo(patientAnswersToQuestions map[int64][]*common.AnswerIntake, questions []*info_intake.Question, context *common.ViewContext, dataApi api.DataAPI) error
}

var genericPopulators []genericPatientViewContextPopulator = make([]genericPatientViewContextPopulator, 0)
var patientQAPopulators map[string]patientQAViewContextPopulator = make(map[string]patientQAViewContextPopulator, 0)

func init() {
	genericPopulators = append(genericPopulators, genericViewContextPopulator(populateAlerts))
	patientQAPopulators[info_intake.QUESTION_TYPE_PHOTO] = qaViewContextPopulator(populatePatientPhotos)
	patientQAPopulators[info_intake.QUESTION_TYPE_MULTIPLE_PHOTO] = qaViewContextPopulator(populatePatientPhotos)
	patientQAPopulators[info_intake.QUESTION_TYPE_SINGLE_PHOTO] = qaViewContextPopulator(populatePatientPhotos)
	patientQAPopulators[info_intake.QUESTION_TYPE_AUTOCOMPLETE] = qaViewContextPopulator(populateAnswersForQuestionsWithSubanswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_MULTIPLE_CHOICE] = qaViewContextPopulator(populateMultipleChoiceAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_SINGLE_ENTRY] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_FREE_TEXT] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_SINGLE_SELECT] = qaViewContextPopulator(populateSingleEntryAnswers)
}

const (
	textReplacementIdentifier = "XXX"
)

type qaViewContextPopulator func([]*common.AnswerIntake, *info_intake.Question, *common.ViewContext, api.DataAPI, api.CloudStorageAPI) error
type genericViewContextPopulator func(map[int64][]*common.AnswerIntake, []*info_intake.Question, *common.ViewContext, api.DataAPI) error

func (q qaViewContextPopulator) populateViewContextWithPatientQA(patientAnswers []*common.AnswerIntake, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, photoStorageService api.CloudStorageAPI) error {
	return q(patientAnswers, question, context, dataApi, photoStorageService)
}

func (g genericViewContextPopulator) populateViewContextWithInfo(patientAnswersToQuestions map[int64][]*common.AnswerIntake, questions []*info_intake.Question, context *common.ViewContext, dataApi api.DataAPI) error {
	return g(patientAnswersToQuestions, questions, context, dataApi)
}

func populateAlerts(patientAnswersToQuestions map[int64][]*common.AnswerIntake, questions []*info_intake.Question, context *common.ViewContext, dataApi api.DataAPI) error {
	questionIdToQuestion := make(map[int64]*info_intake.Question)
	for _, question := range questions {
		questionIdToQuestion[question.QuestionId] = question
	}

	alerts := make([]string, 0)
	// lets go over every answered question
	for questionId, answers := range patientAnswersToQuestions {
		// check if the alert flag is set on the question
		question := questionIdToQuestion[questionId]
		if question.ToAlert {
			switch question.QuestionTypes[0] {

			case info_intake.QUESTION_TYPE_AUTOCOMPLETE:
				// populate the answers to call out in the alert
				enteredAnswers := make([]string, len(answers))
				for i, answer := range answers {

					answerText := answer.AnswerText

					if answerText == "" {
						answerText = answer.AnswerSummary
					}

					if answerText == "" {
						answerText = answer.PotentialAnswer
					}

					enteredAnswers[i] = answerText
				}
				if len(enteredAnswers) > 0 {
					alerts = append(alerts, strings.Replace(question.AlertFormattedText, textReplacementIdentifier, strings.Join(enteredAnswers, ", "), -1))
				}

			case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE, info_intake.QUESTION_TYPE_SINGLE_SELECT:
				selectedAnswers := make([]string, 0)
				for _, potentialAnswer := range question.PotentialAnswers {
					for _, patientAnswer := range answers {
						// populate all the selected answers to show in the alert
						if patientAnswer.PotentialAnswerId.Int64() == potentialAnswer.AnswerId {
							if potentialAnswer.ToAlert {
								if potentialAnswer.AnswerSummary != "" {
									selectedAnswers = append(selectedAnswers, potentialAnswer.AnswerSummary)
								} else {
									selectedAnswers = append(selectedAnswers, potentialAnswer.Answer)
								}
								break
							}
						}
					}
				}
				if len(selectedAnswers) > 0 {
					alerts = append(alerts, strings.Replace(question.AlertFormattedText, textReplacementIdentifier, strings.Join(selectedAnswers, ", "), -1))
				}
			}
		}
	}

	if len(alerts) > 0 {
		context.Set("patient_visit_alerts", alerts)
	} else {
		context.Set("patient_visit_alerts:empty_state_text", "No alerts")
	}

	return nil
}

func populateMultipleChoiceAnswers(patientAnswers []*common.AnswerIntake, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, photoStorageService api.CloudStorageAPI) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	checkedUncheckedItems := make([]info_intake.CheckedUncheckedData, len(question.PotentialAnswers))
	for i, potentialAnswer := range question.PotentialAnswers {
		answerSelected := false

		for _, patientAnswer := range patientAnswers {
			if patientAnswer.PotentialAnswerId.Int64() == potentialAnswer.AnswerId {
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

func populatePatientPhotos(patientAnswers []*common.AnswerIntake, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, photoStorageService api.CloudStorageAPI) error {
	var photos []info_intake.PhotoData
	photoData, ok := context.Get("patient_visit_photos")

	if !ok || photoData == nil {
		photos = make([]info_intake.PhotoData, 0)
	} else {
		photos = photoData.([]info_intake.PhotoData)
	}

	for _, answerIntake := range patientAnswers {
		photos = append(photos, info_intake.PhotoData{
			Title:    answerIntake.PotentialAnswer,
			PhotoUrl: apiservice.GetSignedUrlForAnswer(answerIntake, photoStorageService),
		})
	}

	context.Set("patient_visit_photos", photos)
	return nil
}

func populateSingleEntryAnswers(patientAnswers []*common.AnswerIntake, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, photoStorageService api.CloudStorageAPI) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	if len(patientAnswers) > 1 {
		return fmt.Errorf("Expected just one answer for question %s instead we have  %d", question.QuestionTag, len(patientAnswers))
	}

	answer := patientAnswers[0].AnswerText
	if answer == "" {
		answer = patientAnswers[0].AnswerSummary
	}
	if answer == "" {
		answer = patientAnswers[0].PotentialAnswer
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), answer)
	return nil
}

func populateAnswersForQuestionsWithSubanswers(patientAnswers []*common.AnswerIntake, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, photoStorageService api.CloudStorageAPI) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	data := make([]info_intake.TitleSubtitleSubItemsData, len(patientAnswers))
	for i, patientAnswer := range patientAnswers {

		items := make([]string, len(patientAnswer.SubAnswers))
		for j, subAnswer := range patientAnswer.SubAnswers {
			if subAnswer.AnswerSummary != "" {
				items[j] = subAnswer.AnswerSummary
			} else {
				items[j] = subAnswer.PotentialAnswer
			}
		}

		data[i] = info_intake.TitleSubtitleSubItemsData{
			Title:    patientAnswer.AnswerText,
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

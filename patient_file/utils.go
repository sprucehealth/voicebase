package patient_file

import (
	"fmt"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"net/http"
	"strings"
)

// This interface is used to populate the ViewContext with data pertaining to a single question
type patientQAViewContextPopulator interface {
	populateViewContextWithPatientQA(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, r *http.Request) error
}

// This interface is used to populate the ViewContext with any global data or business logic
type genericPatientViewContextPopulator interface {
	populateViewContextWithInfo(patientAnswersToQuestions map[int64][]common.Answer, questions []*info_intake.Question, context *common.ViewContext, dataApi api.DataAPI) error
}

var genericPopulators []genericPatientViewContextPopulator = make([]genericPatientViewContextPopulator, 0)
var patientQAPopulators map[string]patientQAViewContextPopulator = make(map[string]patientQAViewContextPopulator, 0)

func init() {
	genericPopulators = append(genericPopulators, genericViewContextPopulator(populateAlerts))
	patientQAPopulators[info_intake.QUESTION_TYPE_AUTOCOMPLETE] = qaViewContextPopulator(populateAnswersForQuestionsWithSubanswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_MULTIPLE_CHOICE] = qaViewContextPopulator(populateMultipleChoiceAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_SINGLE_ENTRY] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_FREE_TEXT] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_SINGLE_SELECT] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QUESTION_TYPE_PHOTO_SECTION] = qaViewContextPopulator(populatePatientPhotos)
	patientQAPopulators[info_intake.QUESTION_TYPE_PHOTO] = qaViewContextPopulator(populatePatientPhotos)
}

const (
	textReplacementIdentifier = "XXX"
)

type qaViewContextPopulator func([]common.Answer, *info_intake.Question, *common.ViewContext, api.DataAPI, *http.Request) error
type genericViewContextPopulator func(map[int64][]common.Answer, []*info_intake.Question, *common.ViewContext, api.DataAPI) error

func (q qaViewContextPopulator) populateViewContextWithPatientQA(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, r *http.Request) error {
	return q(patientAnswers, question, context, dataApi, r)
}

func (g genericViewContextPopulator) populateViewContextWithInfo(patientAnswersToQuestions map[int64][]common.Answer, questions []*info_intake.Question, context *common.ViewContext, dataApi api.DataAPI) error {
	return g(patientAnswersToQuestions, questions, context, dataApi)
}

func populateAlerts(patientAnswersToQuestions map[int64][]common.Answer, questions []*info_intake.Question, context *common.ViewContext, dataApi api.DataAPI) error {
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
			switch question.QuestionType {

			case info_intake.QUESTION_TYPE_AUTOCOMPLETE:
				// populate the answers to call out in the alert
				enteredAnswers := make([]string, len(answers))
				for i, answer := range answers {
					if a, ok := answer.(*common.AnswerIntake); ok {
						answerText := a.AnswerText

						if answerText == "" {
							answerText = a.AnswerSummary
						}

						if answerText == "" {
							answerText = a.PotentialAnswer
						}

						enteredAnswers[i] = answerText
					}
				}
				if len(enteredAnswers) > 0 {
					alerts = append(alerts, strings.Replace(question.AlertFormattedText, textReplacementIdentifier, strings.Join(enteredAnswers, ", "), -1))
				}

			case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE, info_intake.QUESTION_TYPE_SINGLE_SELECT:
				selectedAnswers := make([]string, 0)
				for _, potentialAnswer := range question.PotentialAnswers {
					for _, patientAnswer := range answers {
						pAnswer := patientAnswer.(*common.AnswerIntake)
						// populate all the selected answers to show in the alert
						if pAnswer.PotentialAnswerId.Int64() == potentialAnswer.AnswerId {
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

func populateMultipleChoiceAnswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, r *http.Request) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	// if we are dealing with a question that has subquestions defined, populate the context appropriately
	if question.SubQuestionsConfig != nil && (len(question.SubQuestionsConfig.Screens) > 0 || len(question.SubQuestionsConfig.Questions) > 0) {
		return populateAnswersForQuestionsWithSubanswers(patientAnswers, question, context, dataApi, r)
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

func populateSingleEntryAnswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, r *http.Request) error {
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

func populateAnswersForQuestionsWithSubanswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, r *http.Request) error {
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

func populatePatientPhotos(answeredPhotoSections []common.Answer, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI, r *http.Request) error {
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
				PhotoUrl: apiservice.CreatePhotoUrl(photoIntakeSlot.PhotoId, pIntakeSection.Id, common.ClaimerTypePhotoIntakeSection, r.Host),
			}
		}
		items = append(items, item)
	}

	context.Set("patient_visit_photos", items)
	return nil
}

func buildContext(dataApi api.DataAPI, patientVisitLayout *info_intake.InfoIntakeLayout, patientId, patientVisitId int64, req *http.Request) (common.ViewContext, error) {
	questions := apiservice.GetQuestionsInPatientVisitLayout(patientVisitLayout)

	questionIds := apiservice.GetNonPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)
	photoQuestionIds := apiservice.GetPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)

	// get all the answers the patient entered for the questions (note that there may not be an answer for every question)
	patientAnswersForQuestions, err := dataApi.GetPatientAnswersForQuestionsBasedOnQuestionIds(questionIds, patientId, patientVisitId)
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

	context, err := populateContextForRenderingLayout(patientAnswersForQuestions, questions, dataApi, req)
	if err != nil {
		return nil, err
	}

	return context, err
}

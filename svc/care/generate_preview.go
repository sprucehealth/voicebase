package care

import (
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
)

const (
	responsePlaceholder = "<Patient response goes here>"
)

// GenerateVisitLayoutPreview returns a preview of the intake with dummy responses populated.
// This enables the provider to preview a visitLayout as it would look once submited by the patient.
func GenerateVisitLayoutPreview(intake *layout.Intake, review *visitreview.SectionListView) (map[string]interface{}, error) {
	context := visitreview.NewViewContext(nil)
	context.Set("visit_alerts", []string{"<Alerts generated based on patient responses to questions in the visit go here>"})

	for _, section := range intake.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				var builder buildPreviewContextFunc

				if question.SubQuestionsConfig != nil && len(question.SubQuestionsConfig.Screens) > 0 {
					builder = buildPreviewQuestionWithSubanswers
				} else {
					switch question.Type {

					case saml.QuestionTypeAutocomplete,
						saml.QuestionTypeFreeText,
						saml.QuestionTypeSingleEntry:
						builder = buildPreviewQuestionFreeText

					case saml.QuestionTypeMultipleChoice:
						builder = buildPreviewQuestionWithOptions

					case saml.QuestionTypeSingleSelect, saml.QuestionTypeSegmentedControl:
						builder = buildPreviewQuestionWithSingleResponse

					case saml.QuestionTypePhotoSection:
						builder = buildPreviewQuestionWithPhotoSlots

					default:
						return nil, errors.Trace(fmt.Errorf("context builder not found for question of type: %s", question.Type))
					}
				}

				if err := builder(question, context); err != nil {
					return nil, errors.Trace(fmt.Errorf("unable to build context for question '%s': %s", question.ID, err))
				}
			}
		}
	}

	return review.Render(context)
}

type buildPreviewContextFunc func(*layout.Question, *visitreview.ViewContext) error

func buildPreviewQuestionWithOptions(question *layout.Question, context *visitreview.ViewContext) error {
	checkeUncheckedItems := make([]visitreview.CheckedUncheckedData, 0, len(question.PotentialAnswers))
	for _, option := range question.PotentialAnswers {
		checkeUncheckedItems = append(checkeUncheckedItems, visitreview.CheckedUncheckedData{
			Value:     option.Answer,
			IsChecked: false,
		})
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), checkeUncheckedItems)
	return nil
}

func buildPreviewQuestionWithSingleResponse(question *layout.Question, context *visitreview.ViewContext) error {
	possibleOptions := make([]string, 0, len(question.PotentialAnswers))
	for _, option := range question.PotentialAnswers {
		possibleOptions = append(possibleOptions, option.Summary)
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), possibleOptions)
	return nil
}

func buildPreviewQuestionFreeText(question *layout.Question, context *visitreview.ViewContext) error {
	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), responsePlaceholder)
	return nil
}

func buildPreviewQuestionWithSubanswers(question *layout.Question, context *visitreview.ViewContext) error {

	subquestionMap := make(map[string]*layout.Question)
	for _, screen := range question.SubQuestionsConfig.Screens {
		for _, question := range screen.Questions {
			subquestionMap[question.ID] = question
		}
	}

	items := make([]*visitreview.DescriptionContentData, 0, len(subquestionMap))
	for _, screen := range question.SubQuestionsConfig.Screens {
		for _, question := range screen.Questions {
			items = append(items, &visitreview.DescriptionContentData{
				Description: question.Summary,
				Content:     responsePlaceholder,
			})
		}
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), []visitreview.TitleSubItemsDescriptionContentData{
		{
			Title:    responsePlaceholder,
			SubItems: items,
		},
	})

	return nil
}

func buildPreviewQuestionWithPhotoSlots(question *layout.Question, context *visitreview.ViewContext) error {

	context.Set(visitreview.PhotosKey(question.ID), []visitreview.TitlePhotoListData{
		{
			Title: question.Title + ": Photo section",
			Photos: []visitreview.PhotoData{
				{
					Title: "Photo name",
				},
			},
		},
	})

	return nil
}

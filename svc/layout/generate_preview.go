package layout

import (
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/saml"
)

const (
	responsePlaceholder = "<Patient response goes here>"
)

// GenerateVisitLayoutPreview returns a preview of the intake with dummy responses populated.
func GenerateVisitLayoutPreview(intake *Intake, review *visitreview.SectionListView) (map[string]interface{}, error) {
	context := visitreview.NewViewContext(nil)
	context.Set(visitreview.EmptyStateTextKey("visit_alerts"), "<Alerts go here>")

	for _, section := range intake.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				var builder buildContextFunc

				if question.SubQuestionsConfig != nil && len(question.SubQuestionsConfig.Screens) > 0 {
					builder = builderQuestionWithSubanswers
				} else {
					switch question.Type {

					case saml.QuestionTypeAutocomplete,
						saml.QuestionTypeFreeText,
						saml.QuestionTypeSingleEntry:
						builder = builderQuestionFreeText

					case saml.QuestionTypeMultipleChoice:
						builder = builderQuestionWithOptions

					case saml.QuestionTypeSingleSelect, saml.QuestionTypeSegmentedControl:
						builder = builderQuestionWithSingleResponse

					case saml.QuestionTypePhotoSection:
						builder = builderQuestionWithPhotoSlots

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

type buildContextFunc func(*Question, *visitreview.ViewContext) error

func builderQuestionWithOptions(question *Question, context *visitreview.ViewContext) error {
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

func builderQuestionWithSingleResponse(question *Question, context *visitreview.ViewContext) error {
	possibleOptions := make([]string, 0, len(question.PotentialAnswers))
	for _, option := range question.PotentialAnswers {
		possibleOptions = append(possibleOptions, option.Summary)
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), possibleOptions)
	return nil
}

func builderQuestionFreeText(question *Question, context *visitreview.ViewContext) error {
	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), responsePlaceholder)
	return nil
}

func builderQuestionWithSubanswers(question *Question, context *visitreview.ViewContext) error {

	subquestionMap := make(map[string]*Question)
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

func builderQuestionWithPhotoSlots(question *Question, context *visitreview.ViewContext) error {

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

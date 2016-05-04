package client

import (
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/saml"
)

func viewsForQuestionScreen(screen *saml.Screen) ([]visitreview.View, error) {

	views := make([]visitreview.View, 0, len(screen.Questions))

	for _, question := range screen.Questions {

		if question.Details == nil {
			return nil, errors.Trace(fmt.Errorf("unexpected question with no details"))
		}

		if question.SubquestionConfig != nil {
			views = append(views, viewForQuestionWithSubQuestions(question))
			continue
		}

		switch question.Details.Type {
		case saml.QuestionTypeAutocomplete,
			saml.QuestionTypeSingleSelect,
			saml.QuestionTypeSegmentedControl,
			saml.QuestionTypeFreeText:
			views = append(views, viewForQuestionWithSingleAnswer(question))
		case saml.QuestionTypeMultipleChoice:
			views = append(views, viewForQuestionWithMultipleAnswers(question))
		default:
			return nil, errors.Trace(fmt.Errorf("unable to generate view for question type %s", question.Details.Type))
		}

	}

	return views, nil
}

func viewForQuestionWithSubQuestions(question *saml.Question) visitreview.View {
	tag := question.Details.Tag
	return &visitreview.StandardTwoColumnRowView{
		ContentConfig: &visitreview.ContentConfig{
			ViewCondition: visitreview.ViewCondition{
				Op:  visitreview.ConditionKeyExists,
				Key: answersKey(tag),
			},
		},
		LeftView: &visitreview.TitleLabelsList{
			ContentConfig: &visitreview.ContentConfig{
				Key: questionSummaryKey(tag),
			},
		},
		RightView: &visitreview.TitleSubItemsLabelContentItemsList{
			ContentConfig: &visitreview.ContentConfig{
				Key: answersKey(tag),
			},
			EmptyStateView: &visitreview.EmptyLabelView{
				ContentConfig: &visitreview.ContentConfig{
					Key: emptyStateTextKey(tag),
				},
			},
		},
	}
}

func viewForQuestionWithSingleAnswer(question *saml.Question) visitreview.View {
	tag := question.Details.Tag
	return &visitreview.StandardTwoColumnRowView{
		ContentConfig: &visitreview.ContentConfig{
			ViewCondition: visitreview.ViewCondition{
				Op:  visitreview.ConditionKeyExists,
				Key: answersKey(tag),
			},
		},
		LeftView: &visitreview.TitleLabelsList{
			ContentConfig: &visitreview.ContentConfig{
				Key: questionSummaryKey(tag),
			},
		},
		RightView: &visitreview.ContentLabelsList{
			ContentConfig: &visitreview.ContentConfig{
				Key: answersKey(tag),
			},
		},
	}
}

func viewForQuestionWithMultipleAnswers(question *saml.Question) visitreview.View {
	tag := question.Details.Tag
	return &visitreview.StandardTwoColumnRowView{
		ContentConfig: &visitreview.ContentConfig{
			ViewCondition: visitreview.ViewCondition{
				Op:  visitreview.ConditionKeyExists,
				Key: answersKey(tag),
			},
		},
		LeftView: &visitreview.TitleLabelsList{
			ContentConfig: &visitreview.ContentConfig{
				Key: questionSummaryKey(tag),
			},
		},
		RightView: &visitreview.CheckXItemsList{
			ContentConfig: &visitreview.ContentConfig{
				Key: answersKey(tag),
			},
		},
	}
}

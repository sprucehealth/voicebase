package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/settings"
)

func transformVisitToResponse(v *models.Visit, optionalTriage *settings.BooleanValue) *care.Visit {
	return &care.Visit{
		ID:              v.ID.String(),
		Name:            v.Name,
		Submitted:       v.Submitted,
		Triaged:         v.Triaged,
		LayoutVersionID: v.LayoutVersionID,
		EntityID:        v.EntityID,
		OrganizationID:  v.OrganizationID,
		Preferences: &care.Visit_Preference{
			OptionalTriage: optionalTriage.Value,
		},
	}
}

type answerToModelTransformerFunc func(questionID string, answer client.Answer) (*models.Answer, error)

var answerToModelTransformers map[string]answerToModelTransformerFunc

func init() {
	answerToModelTransformers = map[string]answerToModelTransformerFunc{
		"q_type_media_section":     transformMediaSectionAnswerToModel,
		"q_type_free_text":         transformFreeTextAnswerToModel,
		"q_type_single_entry":      transformSingleEntryAnswerToModel,
		"q_type_single_select":     transformSingleSelectAnswerToModel,
		"q_type_multiple_choice":   transformMultipleChoiceAnswerToModel,
		"q_type_segmented_control": transformSegmentedControlAnswerToModel,
		"q_type_autocomplete":      transformAutocompleteAnswerToModel,
	}
}

func transformAnswerToModel(questionID string, answer client.Answer) (*models.Answer, error) {
	transformFunction, ok := answerToModelTransformers[answer.TypeName()]
	if !ok {
		return nil, errors.Trace(fmt.Errorf("cannot find transformer for answer of type %s for question %s", answer.TypeName(), questionID))
	}
	return transformFunction(questionID, answer)
}

func transformMediaSectionAnswerToModel(questionID string, answer client.Answer) (*models.Answer, error) {
	mediaSectionAnswer, ok := answer.(*client.MediaQuestionAnswer)
	if !ok {
		return nil, errors.Trace(fmt.Errorf("expected type MediaQuestionAnswer for answer to question %s but got %T", questionID, answer))
	}

	modelAnswer := &models.Answer{
		QuestionID: questionID,
		Type:       answer.TypeName(),
		Answer: &models.Answer_MediaSection{
			MediaSection: &models.MediaSectionAnswer{
				Sections: make([]*models.MediaSectionAnswer_MediaSectionItem, len(mediaSectionAnswer.Sections)),
			},
		},
	}

	for i, mediaSection := range mediaSectionAnswer.Sections {
		modelAnswer.GetMediaSection().Sections[i] = &models.MediaSectionAnswer_MediaSectionItem{
			Name:  mediaSection.Name,
			Slots: make([]*models.MediaSectionAnswer_MediaSectionItem_MediaSlotItem, len(mediaSection.Slots)),
		}

		for j, mediaSlot := range mediaSection.Slots {
			modelAnswer.GetMediaSection().Sections[i].Slots[j] = &models.MediaSectionAnswer_MediaSectionItem_MediaSlotItem{
				Name:    mediaSlot.Name,
				SlotID:  mediaSlot.SlotID,
				MediaID: mediaSlot.MediaID,
				Type:    "photo", // TODO: Once media service is up, get type from there.
			}
		}
	}

	return modelAnswer, nil
}

func transformFreeTextAnswerToModel(questionID string, answer client.Answer) (*models.Answer, error) {
	freeTextAnswer, ok := answer.(*client.FreeTextQuestionAnswer)
	if !ok {
		return nil, errors.Trace(fmt.Errorf("expected type freeTextAnswer for answer to question %s but got %T", questionID, answer))
	}

	modelAnswer := &models.Answer{
		QuestionID: questionID,
		Type:       answer.TypeName(),
		Answer: &models.Answer_FreeText{
			FreeText: &models.FreeTextAnswer{
				FreeText: freeTextAnswer.Text,
			},
		},
	}

	return modelAnswer, nil
}

func transformSingleEntryAnswerToModel(questionID string, answer client.Answer) (*models.Answer, error) {
	singleEntryAnswer, ok := answer.(*client.SingleEntryQuestionAnswer)
	if !ok {
		return nil, errors.Trace(fmt.Errorf("expected type singleEntryAnswer for answer to question %s but got %T", questionID, answer))
	}

	modelAnswer := &models.Answer{
		QuestionID: questionID,
		Type:       answer.TypeName(),
		Answer: &models.Answer_SingleEntry{
			SingleEntry: &models.SingleEntryAnswer{
				FreeText: singleEntryAnswer.Text,
			},
		},
	}
	return modelAnswer, nil
}

func transformSingleSelectAnswerToModel(questionID string, answer client.Answer) (*models.Answer, error) {
	singleSelectAnswer, ok := answer.(*client.SingleSelectQuestionAnswer)
	if !ok {
		return nil, errors.Trace(fmt.Errorf("expected type singleSelectAnswer for answer to question %s but got %T", questionID, answer))
	}

	modelAnswer := &models.Answer{
		QuestionID: questionID,
		Type:       answer.TypeName(),
		Answer: &models.Answer_SingleSelect{
			SingleSelect: &models.SingleSelectAnswer{
				SelectedAnswer: &models.AnswerOption{
					ID:       singleSelectAnswer.PotentialAnswer.ID,
					FreeText: singleSelectAnswer.PotentialAnswer.Text,
				},
			},
		},
	}

	return modelAnswer, nil
}

func transformSegmentedControlAnswerToModel(questionID string, answer client.Answer) (*models.Answer, error) {
	segmentedControlAnswer, ok := answer.(*client.SegmentedControlQuestionAnswer)
	if !ok {
		return nil, errors.Trace(fmt.Errorf("expected type segmentedControlAnswer for answer to question %s but got %T", questionID, answer))
	}

	modelAnswer := &models.Answer{
		QuestionID: questionID,
		Type:       answer.TypeName(),
		Answer: &models.Answer_SegmentedControl{
			SegmentedControl: &models.SegmentedControlAnswer{
				SelectedAnswer: &models.AnswerOption{
					ID:       segmentedControlAnswer.PotentialAnswer.ID,
					FreeText: segmentedControlAnswer.PotentialAnswer.Text,
				},
			},
		},
	}

	return modelAnswer, nil
}

func transformMultipleChoiceAnswerToModel(questionID string, answer client.Answer) (*models.Answer, error) {
	multipleChoiceAnswer, ok := answer.(*client.MultipleChoiceQuestionAnswer)
	if !ok {
		return nil, errors.Trace(fmt.Errorf("expected type multipleChoiceAnswer for answer to question %s but got %T", questionID, answer))
	}

	modelAnswer := &models.Answer{
		QuestionID: questionID,
		Type:       answer.TypeName(),
		Answer: &models.Answer_MultipleChoice{
			MultipleChoice: &models.MultipleChoiceAnswer{
				SelectedAnswers: make([]*models.AnswerOption, len(multipleChoiceAnswer.PotentialAnswers)),
			},
		},
	}

	for i, potentialAnswer := range multipleChoiceAnswer.PotentialAnswers {
		modelAnswer.GetMultipleChoice().SelectedAnswers[i] = &models.AnswerOption{
			ID:         potentialAnswer.ID,
			FreeText:   potentialAnswer.Text,
			SubAnswers: make(map[string]*models.Answer, len(potentialAnswer.Subanswers)),
		}

		var err error
		for subquestionID, subanswer := range potentialAnswer.Subanswers {
			modelAnswer.GetMultipleChoice().SelectedAnswers[i].SubAnswers[subquestionID], err = transformAnswerToModel(subquestionID, subanswer)
			if err != nil {
				return nil, errors.Trace(fmt.Errorf("unable to transform subanswer %s for answer %s to question %s: %s", subanswer.TypeName(), potentialAnswer.ID, questionID, err))
			}
		}
	}

	return modelAnswer, nil
}

func transformAutocompleteAnswerToModel(questionID string, answer client.Answer) (*models.Answer, error) {
	autocompleteAnswer, ok := answer.(*client.AutocompleteQuestionAnswer)
	if !ok {
		return nil, errors.Trace(fmt.Errorf("expected type autocompleteAnswer for answer to question %s but got %T", questionID, answer))
	}

	modelAnswer := &models.Answer{
		QuestionID: questionID,
		Type:       answer.TypeName(),
		Answer: &models.Answer_Autocomplete{
			Autocomplete: &models.AutocompleteAnswer{
				Items: make([]*models.AutocompleteAnswerItem, len(autocompleteAnswer.Answers)),
			},
		},
	}

	for i, item := range autocompleteAnswer.Answers {
		modelAnswer.GetAutocomplete().Items[i] = &models.AutocompleteAnswerItem{
			Answer:     item.Text,
			SubAnswers: make(map[string]*models.Answer, len(item.Subanswers)),
		}

		var err error
		for subquestionID, subanswer := range item.Subanswers {
			modelAnswer.GetAutocomplete().Items[i].SubAnswers[subquestionID], err = transformAnswerToModel(subquestionID, subanswer)
			if err != nil {
				return nil, errors.Trace(fmt.Errorf("unable to transform subanswer %s to question %s: %s", subanswer.TypeName(), questionID, err))
			}
		}
	}
	return modelAnswer, nil
}

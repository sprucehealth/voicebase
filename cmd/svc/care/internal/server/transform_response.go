package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
)

type answerModelToResponseTransformerFunc func(answer *models.Answer) (client.Answer, error)

var answerModelToResponseTransformers map[string]answerModelToResponseTransformerFunc

func init() {
	answerModelToResponseTransformers = map[string]answerModelToResponseTransformerFunc{
		"q_type_photo_section":     transformPhotoSectionToResponse,
		"q_type_free_text":         transformFreeTextAnswerToResponse,
		"q_type_single_entry":      transformSingleEntryAnswerToResponse,
		"q_type_single_select":     transformSingleSelectAnswerToResponse,
		"q_type_multiple_choice":   transformMultipleChoiceAnswerToResponse,
		"q_type_segmented_control": transformSegmentedControlAnswerToResponse,
		"q_type_autocomplete":      transformAutocompleteAnswerToResponse,
	}
}

func transformAnswerModelToResponse(answer *models.Answer) (client.Answer, error) {
	transformFunc, ok := answerModelToResponseTransformers[answer.Type]
	if !ok {
		return nil, errors.Trace(fmt.Errorf("unable to find a response transformer for answer type %s", answer.Type))
	}

	return transformFunc(answer)
}

func transformPhotoSectionToResponse(answer *models.Answer) (client.Answer, error) {
	if answer.GetPhotoSection() == nil {
		return nil, errors.Trace(fmt.Errorf("expected photo section to be populated for answer but it wasnt"))
	}
	photoSectionAnswer := &client.PhotoQuestionAnswer{
		Type:          answer.Type,
		PhotoSections: make([]*client.PhotoSectionItem, len(answer.GetPhotoSection().Sections)),
	}

	for i, photoSection := range answer.GetPhotoSection().Sections {
		photoSectionAnswer.PhotoSections[i] = &client.PhotoSectionItem{
			Name:  photoSection.Name,
			Slots: make([]*client.PhotoSlotItem, len(photoSection.Slots)),
		}

		for j, photoSlot := range photoSection.Slots {
			photoSectionAnswer.PhotoSections[i].Slots[j] = &client.PhotoSlotItem{
				Name:    photoSlot.Name,
				SlotID:  photoSlot.SlotID,
				PhotoID: photoSlot.MediaID,
			}
		}
	}
	return photoSectionAnswer, nil
}

func transformFreeTextAnswerToResponse(answer *models.Answer) (client.Answer, error) {
	if answer.GetFreeText() == nil {
		return nil, errors.Trace(fmt.Errorf("expected free text answer to be populated for answer but it wasnt"))
	}

	return &client.FreeTextQuestionAnswer{
		Type: answer.Type,
		Text: answer.GetFreeText().FreeText,
	}, nil
}

func transformSingleEntryAnswerToResponse(answer *models.Answer) (client.Answer, error) {
	if answer.GetSingleEntry() == nil {
		return nil, errors.Trace(fmt.Errorf("expected single entry answer to be populated for answer but it wasnt"))
	}

	return &client.SingleEntryQuestionAnswer{
		Type: answer.Type,
		Text: answer.GetSingleEntry().FreeText,
	}, nil
}

func transformSingleSelectAnswerToResponse(answer *models.Answer) (client.Answer, error) {
	if answer.GetSingleSelect() == nil {
		return nil, errors.Trace(fmt.Errorf("expected single select answer to be populated for answer but it wasnt"))
	}

	return &client.SingleSelectQuestionAnswer{
		Type: answer.Type,
		PotentialAnswer: &client.PotentialAnswerItem{
			ID:   answer.GetSingleSelect().SelectedAnswer.ID,
			Text: answer.GetSingleSelect().SelectedAnswer.FreeText,
		},
	}, nil
}

func transformSegmentedControlAnswerToResponse(answer *models.Answer) (client.Answer, error) {
	if answer.GetSegmentedControl() == nil {
		return nil, errors.Trace(fmt.Errorf("expected segmented control answer to be populated for answer but it wasnt"))
	}

	return &client.SegmentedControlQuestionAnswer{
		Type: answer.Type,
		PotentialAnswer: &client.PotentialAnswerItem{
			ID:   answer.GetSegmentedControl().SelectedAnswer.ID,
			Text: answer.GetSegmentedControl().SelectedAnswer.FreeText,
		},
	}, nil
}

func transformMultipleChoiceAnswerToResponse(answer *models.Answer) (client.Answer, error) {
	if answer.GetMultipleChoice() == nil {
		return nil, errors.Trace(fmt.Errorf("expected multiple choice answer to be populated for answer but it wasnt"))
	}

	multipleChoiceAnswer := &client.MultipleChoiceQuestionAnswer{
		Type:             answer.Type,
		PotentialAnswers: make([]*client.PotentialAnswerItem, len(answer.GetMultipleChoice().SelectedAnswers)),
	}

	for i, selectedAnswer := range answer.GetMultipleChoice().SelectedAnswers {
		multipleChoiceAnswer.PotentialAnswers[i] = &client.PotentialAnswerItem{
			ID:         selectedAnswer.ID,
			Text:       selectedAnswer.FreeText,
			Subanswers: make(map[string]client.Answer, len(selectedAnswer.SubAnswers)),
		}

		for subquestionID, subanswer := range selectedAnswer.SubAnswers {
			var err error
			multipleChoiceAnswer.PotentialAnswers[i].Subanswers[subquestionID], err = transformAnswerModelToResponse(subanswer)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	return multipleChoiceAnswer, nil
}

func transformAutocompleteAnswerToResponse(answer *models.Answer) (client.Answer, error) {
	if answer.GetAutocomplete() == nil {
		return nil, errors.Trace(fmt.Errorf("expected autocomplete answer to be populated for answer but it wasnt"))
	}

	autocompleteAnswer := &client.AutocompleteQuestionAnswer{
		Type:    answer.Type,
		Answers: make([]*client.AutocompleteItem, len(answer.GetAutocomplete().Items)),
	}
	for i, item := range answer.GetAutocomplete().Items {
		autocompleteAnswer.Answers[i] = &client.AutocompleteItem{
			Text:       item.Answer,
			Subanswers: make(map[string]client.Answer, len(item.SubAnswers)),
		}

		for subquestionID, subanswer := range item.SubAnswers {
			var err error
			autocompleteAnswer.Answers[i].Subanswers[subquestionID], err = transformAnswerModelToResponse(subanswer)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	return autocompleteAnswer, nil
}

package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/care"
)

type answerModelToSVCResponseTransformerFunc func(answer *models.Answer) (*care.Answer, error)

var answerModelToSVCResponseTransformers map[string]answerModelToSVCResponseTransformerFunc

func init() {
	answerModelToSVCResponseTransformers = map[string]answerModelToSVCResponseTransformerFunc{
		"q_type_media_section":     transformMediaSectionToSVCResponse,
		"q_type_free_text":         transformFreeTextAnswerToSVCResponse,
		"q_type_single_entry":      transformSingleEntryAnswerToSVCResponse,
		"q_type_single_select":     transformSingleSelectAnswerToSVCResponse,
		"q_type_segmented_control": transformSegmentedControlAnswerToSVCResponse,
		"q_type_multiple_choice":   transformMultipleChoiceAnswerToSVCResponse,
		"q_type_autocomplete":      transformAutocompleteAnswerToSVCResponse,
	}
}

func transformAnswerModelToSVCResponse(answer *models.Answer) (*care.Answer, error) {
	transformFunc, ok := answerModelToSVCResponseTransformers[answer.Type]
	if !ok {
		return nil, errors.Trace(fmt.Errorf("unable to find a response transformer for answer type %s", answer.Type))
	}

	return transformFunc(answer)
}

func transformMediaSectionToSVCResponse(answer *models.Answer) (*care.Answer, error) {
	if answer.GetMediaSection() == nil {
		return nil, errors.Trace(fmt.Errorf("expected media section to be populated for answer but it wasnt"))
	}

	mediaSectionAnswer := &care.Answer{
		QuestionID: answer.QuestionID,
		Answer: &care.Answer_MediaSection{
			MediaSection: &care.MediaSectionAnswer{
				Sections: make([]*care.MediaSectionAnswer_MediaSectionItem, len(answer.GetMediaSection().Sections)),
			},
		},
	}

	for i, mediaSection := range answer.GetMediaSection().Sections {
		mediaSectionAnswer.GetMediaSection().Sections[i] = &care.MediaSectionAnswer_MediaSectionItem{
			Name:  mediaSection.Name,
			Slots: make([]*care.MediaSectionAnswer_MediaSectionItem_MediaSlotItem, len(mediaSection.Slots)),
		}

		for j, mediaSlot := range mediaSection.Slots {
			mediaSectionAnswer.GetMediaSection().Sections[i].Slots[j] = &care.MediaSectionAnswer_MediaSectionItem_MediaSlotItem{
				Name:    mediaSlot.Name,
				SlotID:  mediaSlot.SlotID,
				MediaID: mediaSlot.MediaID,
				URL:     "https://placekitten.com/600/800", //TODO
				Type:    mediaSlot.Type,
			}
		}
	}
	return mediaSectionAnswer, nil
}

func transformFreeTextAnswerToSVCResponse(answer *models.Answer) (*care.Answer, error) {
	if answer.GetFreeText() == nil {
		return nil, errors.Trace(fmt.Errorf("expected free text answer to be populated for answer but it wasnt"))
	}

	return &care.Answer{
		QuestionID: answer.QuestionID,
		Answer: &care.Answer_FreeText{
			FreeText: &care.FreeTextAnswer{
				FreeText: answer.GetFreeText().FreeText,
			},
		},
	}, nil
}

func transformSingleEntryAnswerToSVCResponse(answer *models.Answer) (*care.Answer, error) {
	if answer.GetSingleEntry() == nil {
		return nil, errors.Trace(fmt.Errorf("expected single entry answer to be populated for answer but it wasnt"))
	}

	return &care.Answer{
		QuestionID: answer.QuestionID,
		Answer: &care.Answer_SingleEntry{
			SingleEntry: &care.SingleEntryAnswer{
				FreeText: answer.GetSingleEntry().FreeText,
			},
		},
	}, nil
}

func transformSingleSelectAnswerToSVCResponse(answer *models.Answer) (*care.Answer, error) {
	if answer.GetSingleSelect() == nil {
		return nil, errors.Trace(fmt.Errorf("expected single select answer to be populated for answer but it wasnt"))
	}

	return &care.Answer{
		QuestionID: answer.QuestionID,
		Answer: &care.Answer_SingleSelect{
			SingleSelect: &care.SingleSelectAnswer{
				SelectedAnswer: &care.AnswerOption{
					ID:       answer.GetSingleSelect().SelectedAnswer.ID,
					FreeText: answer.GetSingleSelect().SelectedAnswer.FreeText,
				},
			},
		},
	}, nil
}

func transformSegmentedControlAnswerToSVCResponse(answer *models.Answer) (*care.Answer, error) {
	if answer.GetSegmentedControl() == nil {
		return nil, errors.Trace(fmt.Errorf("expected segmented control answer to be populated for answer but it wasnt"))
	}

	return &care.Answer{
		QuestionID: answer.QuestionID,
		Answer: &care.Answer_SegmentedControl{
			SegmentedControl: &care.SegmentedControlAnswer{
				SelectedAnswer: &care.AnswerOption{
					ID:       answer.GetSegmentedControl().SelectedAnswer.ID,
					FreeText: answer.GetSegmentedControl().SelectedAnswer.FreeText,
				},
			},
		},
	}, nil
}

func transformMultipleChoiceAnswerToSVCResponse(answer *models.Answer) (*care.Answer, error) {
	if answer.GetMultipleChoice() == nil {
		return nil, errors.Trace(fmt.Errorf("expected multiple choice answer to be populated for answer but it wasnt"))
	}

	multipleChoiceAnswer := &care.Answer{
		QuestionID: answer.QuestionID,
		Answer: &care.Answer_MultipleChoice{
			MultipleChoice: &care.MultipleChoiceAnswer{
				SelectedAnswers: make([]*care.AnswerOption, len(answer.GetMultipleChoice().SelectedAnswers)),
			},
		},
	}

	for i, selectedAnswer := range answer.GetMultipleChoice().SelectedAnswers {
		multipleChoiceAnswer.GetMultipleChoice().SelectedAnswers[i] = &care.AnswerOption{
			ID:         selectedAnswer.ID,
			FreeText:   selectedAnswer.FreeText,
			SubAnswers: make(map[string]*care.Answer, len(selectedAnswer.SubAnswers)),
		}

		for subquestionID, subanswer := range selectedAnswer.SubAnswers {
			var err error
			multipleChoiceAnswer.GetMultipleChoice().SelectedAnswers[i].SubAnswers[subquestionID], err = transformAnswerModelToSVCResponse(subanswer)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	return multipleChoiceAnswer, nil
}

func transformAutocompleteAnswerToSVCResponse(answer *models.Answer) (*care.Answer, error) {
	if answer.GetAutocomplete() == nil {
		return nil, errors.Trace(fmt.Errorf("expected autocomplete answer to be populated for answer but it wasnt"))
	}

	autocompleteAnswer := &care.Answer{
		QuestionID: answer.QuestionID,
		Answer: &care.Answer_Autocomplete{
			Autocomplete: &care.AutocompleteAnswer{
				Items: make([]*care.AutocompleteAnswerItem, len(answer.GetAutocomplete().Items)),
			},
		},
	}
	for i, item := range answer.GetAutocomplete().Items {
		autocompleteAnswer.GetAutocomplete().Items[i] = &care.AutocompleteAnswerItem{
			Answer:     item.Answer,
			SubAnswers: make(map[string]*care.Answer, len(item.SubAnswers)),
		}

		for subquestionID, subanswer := range item.SubAnswers {
			var err error
			autocompleteAnswer.GetAutocomplete().Items[i].SubAnswers[subquestionID], err = transformAnswerModelToSVCResponse(subanswer)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	return autocompleteAnswer, nil
}

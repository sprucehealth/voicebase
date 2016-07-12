package server

import (
	"fmt"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
)

type answerModelToSVCResponseTransformer interface {
	transform(answer *models.Answer) (*care.Answer, error)
}

func transformAnswerModelToSVCResponse(answer *models.Answer, mediaClient media.MediaClient) (*care.Answer, error) {
	var t answerModelToSVCResponseTransformer

	switch answer.Type {
	case layout.QuestionTypeMediaSection:
		t = &mediaSectionToSVCResponseTransformer{
			mediaClient: mediaClient,
		}
	case layout.QuestionTypeFreeText:
		t = &freeTextAnswerToSVCResponseTransformer{}
	case layout.QuestionTypeSingleEntry:
		t = &singleEntryToSVCResponseTransfomer{}
	case layout.QuestionTypeSingleSelect:
		t = &singleSelectToSVCResponseTransformer{}
	case layout.QuestionTypeMultipleChoice:
		t = &multipleChoiceToSVCResponseTransformer{
			mediaClient: mediaClient,
		}
	case layout.QuestionTypeSegmentedControl:
		t = &segmentedControlToSVCResponseTransformer{}
	case layout.QuestionTypeAutoComplete:
		t = &autocompleteAnswerTOSVCResponseTransformer{
			mediaClient: mediaClient,
		}
	default:
		return nil, errors.Trace(fmt.Errorf("cannot find transformer for answer of type %s for question %s", answer.Type, answer.QuestionID))
	}

	return t.transform(answer)
}

type mediaSectionToSVCResponseTransformer struct {
	mediaClient media.MediaClient
}

func (m *mediaSectionToSVCResponseTransformer) transform(answer *models.Answer) (*care.Answer, error) {
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

	mediaIDs := make([]string, 0, len(answer.GetMediaSection().Sections)*3)
	slotMap := make(map[string]*care.MediaSectionAnswer_MediaSectionItem_MediaSlotItem)
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
			}
			mediaIDs = append(mediaIDs, mediaSlot.MediaID)
			slotMap[mediaSlot.MediaID] = mediaSectionAnswer.GetMediaSection().Sections[i].Slots[j]
		}
	}

	res, err := m.mediaClient.MediaInfos(context.Background(), &media.MediaInfosRequest{
		MediaIDs: mediaIDs,
	})
	if err != nil {
		return nil, errors.Trace(fmt.Errorf("Unable to get media info for answer to question %s: %s", answer.QuestionID, err))
	}

	for _, mediaInfo := range res.MediaInfos {
		mediaSlot, ok := slotMap[mediaInfo.ID]
		if !ok {
			return nil, errors.Trace(fmt.Errorf("Unable to find slot that media %s maps to for answer to question %s", mediaInfo.ID, answer.QuestionID))
		}
		mediaSlot.URL = mediaInfo.URL
		mediaSlot.ThumbnailURL = mediaInfo.ThumbURL
		var mediaType care.MediaType
		switch mediaInfo.MIME.Type {
		case "image":
			mediaType = care.MediaType_IMAGE
		case "video":
			mediaType = care.MediaType_VIDEO
		default:
			return nil, errors.Trace(fmt.Errorf("Unknown media type for %s", mediaInfo.ID))
		}
		mediaSlot.Type = mediaType
		delete(slotMap, mediaInfo.ID)
	}

	// there should be no slot left for which we were unable to find the media object
	if len(slotMap) > 0 {
		return nil, errors.Trace(fmt.Errorf("mediaIDs not found for %+v for question %s", slotMap, answer.QuestionID))
	}

	return mediaSectionAnswer, nil
}

type freeTextAnswerToSVCResponseTransformer struct{}

func (f *freeTextAnswerToSVCResponseTransformer) transform(answer *models.Answer) (*care.Answer, error) {
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

type singleEntryToSVCResponseTransfomer struct{}

func (s *singleEntryToSVCResponseTransfomer) transform(answer *models.Answer) (*care.Answer, error) {
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

type singleSelectToSVCResponseTransformer struct{}

func (s *singleSelectToSVCResponseTransformer) transform(answer *models.Answer) (*care.Answer, error) {
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

type segmentedControlToSVCResponseTransformer struct{}

func (s *segmentedControlToSVCResponseTransformer) transform(answer *models.Answer) (*care.Answer, error) {
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

type multipleChoiceToSVCResponseTransformer struct {
	mediaClient media.MediaClient
}

func (m *multipleChoiceToSVCResponseTransformer) transform(answer *models.Answer) (*care.Answer, error) {
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
			multipleChoiceAnswer.GetMultipleChoice().SelectedAnswers[i].SubAnswers[subquestionID], err = transformAnswerModelToSVCResponse(subanswer, m.mediaClient)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	return multipleChoiceAnswer, nil
}

type autocompleteAnswerTOSVCResponseTransformer struct {
	mediaClient media.MediaClient
}

func (a *autocompleteAnswerTOSVCResponseTransformer) transform(answer *models.Answer) (*care.Answer, error) {
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
			autocompleteAnswer.GetAutocomplete().Items[i].SubAnswers[subquestionID], err = transformAnswerModelToSVCResponse(subanswer, a.mediaClient)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	return autocompleteAnswer, nil
}

package server

import (
	"fmt"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/settings"
)

func transformVisitToResponse(v *models.Visit, optionalTriage *settings.BooleanValue) *care.Visit {
	var submittedTimestamp uint64
	if v.SubmittedTimestamp != nil {
		submittedTimestamp = uint64(v.SubmittedTimestamp.Unix())
	}

	return &care.Visit{
		ID:                 v.ID.String(),
		Name:               v.Name,
		Submitted:          v.Submitted,
		SubmittedTimestamp: submittedTimestamp,
		Triaged:            v.Triaged,
		LayoutVersionID:    v.LayoutVersionID,
		EntityID:           v.EntityID,
		OrganizationID:     v.OrganizationID,
		Preferences: &care.Visit_Preference{
			OptionalTriage: optionalTriage.Value,
		},
	}
}

type answerToModelTransformer interface {
	transform(questionID string, answer client.Answer) (*models.Answer, error)
}

func transformAnswerToModel(questionID string, answer client.Answer, mediaClient media.MediaClient) (*models.Answer, error) {

	var t answerToModelTransformer

	switch answer.TypeName() {
	case layout.QuestionTypeMediaSection:
		t = &mediaSectionTransformer{
			mediaClient: mediaClient,
		}
	case layout.QuestionTypeFreeText:
		t = &freeTextTransformer{}
	case layout.QuestionTypeSingleEntry:
		t = &singleEntryTransformer{}
	case layout.QuestionTypeSingleSelect:
		t = &singleSelectTransformer{}
	case layout.QuestionTypeMultipleChoice:
		t = &multipleChoiceAnswerTransformer{
			mediaClient: mediaClient,
		}
	case layout.QuestionTypeSegmentedControl:
		t = &segmentedControlTransformer{}
	case layout.QuestionTypeAutoComplete:
		t = &autocompleteAnswerTransformer{
			mediaClient: mediaClient,
		}
	default:
		return nil, errors.Trace(fmt.Errorf("cannot find transformer for answer of type %s for question %s", answer.TypeName(), questionID))
	}

	return t.transform(questionID, answer)
}

type mediaSectionTransformer struct {
	mediaClient media.MediaClient
}

func (m *mediaSectionTransformer) transform(questionID string, answer client.Answer) (*models.Answer, error) {
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

	slotMap := make(map[string]*models.MediaSectionAnswer_MediaSectionItem_MediaSlotItem)
	mediaIDs := make([]string, 0, len(mediaSectionAnswer.Sections)*3)
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
			}
			slotMap[mediaSlot.MediaID] = modelAnswer.GetMediaSection().Sections[i].Slots[j]
			mediaIDs = append(mediaIDs, mediaSlot.MediaID)
		}
	}

	if len(mediaIDs) > 0 {
		res, err := m.mediaClient.MediaInfos(context.Background(), &media.MediaInfosRequest{
			MediaIDs: mediaIDs,
		})
		if err != nil {
			return nil, errors.Trace(fmt.Errorf("Unable to transform answer for %s: %s", questionID, err))
		}

		for _, mediaInfo := range res.MediaInfos {
			slot, ok := slotMap[mediaInfo.ID]
			if !ok {
				return nil, errors.Trace(fmt.Errorf("media returned for slot that doesn't exist for question %s:%s", questionID, err))
			}
			var mediaType models.MediaType
			switch mediaInfo.MIME.Type {
			case "image":
				mediaType = models.MediaType_IMAGE
			case "video":
				mediaType = models.MediaType_VIDEO
			default:
				return nil, errors.Trace(fmt.Errorf("Unknown media type for %s", mediaInfo.ID))
			}
			slot.Type = mediaType
			delete(slotMap, mediaInfo.ID)
		}
	}

	// there should be no slot left for which we were unable to find the media object
	if len(slotMap) > 0 {
		return nil, errors.Trace(fmt.Errorf("mediaIDs not found for %+v for question %s", slotMap, questionID))
	}

	return modelAnswer, nil
}

type freeTextTransformer struct{}

func (f *freeTextTransformer) transform(questionID string, answer client.Answer) (*models.Answer, error) {
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

type singleEntryTransformer struct{}

func (s *singleEntryTransformer) transform(questionID string, answer client.Answer) (*models.Answer, error) {
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

type singleSelectTransformer struct{}

func (s *singleSelectTransformer) transform(questionID string, answer client.Answer) (*models.Answer, error) {
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

type segmentedControlTransformer struct{}

func (s *segmentedControlTransformer) transform(questionID string, answer client.Answer) (*models.Answer, error) {
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

type multipleChoiceAnswerTransformer struct {
	mediaClient media.MediaClient
}

func (m *multipleChoiceAnswerTransformer) transform(questionID string, answer client.Answer) (*models.Answer, error) {
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
			modelAnswer.GetMultipleChoice().SelectedAnswers[i].SubAnswers[subquestionID], err = transformAnswerToModel(subquestionID, subanswer, m.mediaClient)
			if err != nil {
				return nil, errors.Trace(fmt.Errorf("unable to transform subanswer %s for answer %s to question %s: %s", subanswer.TypeName(), potentialAnswer.ID, questionID, err))
			}
		}
	}

	return modelAnswer, nil
}

type autocompleteAnswerTransformer struct {
	mediaClient media.MediaClient
}

func (a *autocompleteAnswerTransformer) transform(questionID string, answer client.Answer) (*models.Answer, error) {
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
			modelAnswer.GetAutocomplete().Items[i].SubAnswers[subquestionID], err = transformAnswerToModel(subquestionID, subanswer, a.mediaClient)
			if err != nil {
				return nil, errors.Trace(fmt.Errorf("unable to transform subanswer %s to question %s: %s", subanswer.TypeName(), questionID, err))
			}
		}
	}
	return modelAnswer, nil
}

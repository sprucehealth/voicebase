package server

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
)

type answerModelToResponseTransformer interface {
	transform(answer *models.Answer) (client.Answer, error)
}

func transformAnswerModelToResponse(answer *models.Answer, mediaClient media.MediaClient) (client.Answer, error) {
	var t answerModelToResponseTransformer

	switch answer.Type {
	case layout.QuestionTypeMediaSection:
		t = &mediaSectionToResponseTransformer{
			mediaClient: mediaClient,
		}
	case layout.QuestionTypeFreeText:
		t = &freeTextAnswerToResponseTransformer{}
	case layout.QuestionTypeSingleEntry:
		t = &singleEntryToResponseTransformer{}
	case layout.QuestionTypeSingleSelect:
		t = &singleSelectToResponseTransformer{}
	case layout.QuestionTypeMultipleChoice:
		t = &multipleChoiceToResponseTransformer{
			mediaClient: mediaClient,
		}
	case layout.QuestionTypeSegmentedControl:
		t = &segmentedControlToResponseTransformer{}
	case layout.QuestionTypeAutoComplete:
		t = &autocompleteToResponseTransformer{
			mediaClient: mediaClient,
		}
	default:
		return nil, errors.Trace(fmt.Errorf("cannot find transformer for answer of type %s for question %s", answer.Type, answer.QuestionID))
	}

	return t.transform(answer)
}

type mediaSectionToResponseTransformer struct {
	mediaClient media.MediaClient
}

func (m *mediaSectionToResponseTransformer) transform(answer *models.Answer) (client.Answer, error) {
	if answer.GetMediaSection() == nil {
		return nil, errors.Trace(fmt.Errorf("expected media section to be populated for answer but it wasnt"))
	}
	mediaSectionAnswer := &client.MediaQuestionAnswer{
		Type:     answer.Type,
		Sections: make([]*client.MediaSectionItem, len(answer.GetMediaSection().Sections)),
	}

	mediaIDs := make([]string, 0, len(answer.GetMediaSection().Sections)*3)
	slotMap := make(map[string]*client.MediaSlotItem)
	for i, mediaSection := range answer.GetMediaSection().Sections {
		mediaSectionAnswer.Sections[i] = &client.MediaSectionItem{
			Name:  mediaSection.Name,
			Slots: make([]*client.MediaSlotItem, len(mediaSection.Slots)),
		}

		for j, mediaSlot := range mediaSection.Slots {
			mediaIDs = append(mediaIDs, mediaSlot.MediaID)

			mediaSectionAnswer.Sections[i].Slots[j] = &client.MediaSlotItem{
				Name:    mediaSlot.Name,
				SlotID:  mediaSlot.SlotID,
				MediaID: mediaSlot.MediaID,
			}
			slotMap[mediaSlot.MediaID] = mediaSectionAnswer.Sections[i].Slots[j]
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
		mediaSlot.Type = mediaInfo.MIME.Type
		delete(slotMap, mediaInfo.ID)
	}

	// there should be no slot left for which we were unable to find the media object
	if len(slotMap) > 0 {
		return nil, errors.Trace(fmt.Errorf("mediaIDs not found for %+v for question %s", slotMap, answer.QuestionID))
	}

	return mediaSectionAnswer, nil
}

type freeTextAnswerToResponseTransformer struct{}

func (f *freeTextAnswerToResponseTransformer) transform(answer *models.Answer) (client.Answer, error) {
	if answer.GetFreeText() == nil {
		return nil, errors.Trace(fmt.Errorf("expected free text answer to be populated for answer but it wasnt"))
	}

	return &client.FreeTextQuestionAnswer{
		Type: answer.Type,
		Text: answer.GetFreeText().FreeText,
	}, nil
}

type singleEntryToResponseTransformer struct {
}

func (s *singleEntryToResponseTransformer) transform(answer *models.Answer) (client.Answer, error) {
	if answer.GetSingleEntry() == nil {
		return nil, errors.Trace(fmt.Errorf("expected single entry answer to be populated for answer but it wasnt"))
	}

	return &client.SingleEntryQuestionAnswer{
		Type: answer.Type,
		Text: answer.GetSingleEntry().FreeText,
	}, nil
}

type singleSelectToResponseTransformer struct{}

func (s *singleSelectToResponseTransformer) transform(answer *models.Answer) (client.Answer, error) {
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

type segmentedControlToResponseTransformer struct{}

func (s *segmentedControlToResponseTransformer) transform(answer *models.Answer) (client.Answer, error) {
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

type multipleChoiceToResponseTransformer struct {
	mediaClient media.MediaClient
}

func (m *multipleChoiceToResponseTransformer) transform(answer *models.Answer) (client.Answer, error) {
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
			multipleChoiceAnswer.PotentialAnswers[i].Subanswers[subquestionID], err = transformAnswerModelToResponse(subanswer, m.mediaClient)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	return multipleChoiceAnswer, nil
}

type autocompleteToResponseTransformer struct {
	mediaClient media.MediaClient
}

func (a *autocompleteToResponseTransformer) transform(answer *models.Answer) (client.Answer, error) {
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
			autocompleteAnswer.Answers[i].Subanswers[subquestionID], err = transformAnswerModelToResponse(subanswer, a.mediaClient)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	return autocompleteAnswer, nil
}

func transformCarePlanToResponse(cp *models.CarePlan) (*care.CarePlan, error) {
	cpr := &care.CarePlan{
		ID:               cp.ID.String(),
		Name:             cp.Name,
		CreatedTimestamp: uint64(cp.Created.Unix()),
		ParentID:         cp.ParentID,
		CreatorID:        cp.CreatorID,
	}
	if cp.Submitted != nil {
		cpr.Submitted = true
		cpr.SubmittedTimestamp = uint64(cp.Submitted.Unix())
	}

	cpr.Instructions = make([]*care.CarePlanInstruction, len(cp.Instructions))
	for i, ins := range cp.Instructions {
		cpr.Instructions[i] = &care.CarePlanInstruction{
			Title: ins.Title,
			Steps: ins.Steps,
		}
	}

	cpr.Treatments = make([]*care.CarePlanTreatment, len(cp.Treatments))
	for i, t := range cp.Treatments {
		var availability care.CarePlanTreatment_Availability
		switch t.Availability {
		case models.TreatmentAvailabilityUnknown:
			availability = care.CarePlanTreatment_UNKNOWN
		case models.TreatmentAvailabilityOTC:
			availability = care.CarePlanTreatment_OTC
		case models.TreatmentAvailabilityRx:
			availability = care.CarePlanTreatment_RX
		default:
			return nil, errors.Trace(fmt.Errorf("unknown treatment availability '%s' while transforming treatment '%s' to response", t.Availability, t.ID))
		}
		cpr.Treatments[i] = &care.CarePlanTreatment{
			EPrescribe:           t.EPrescribe,
			Availability:         availability,
			Name:                 t.Name,
			Route:                t.Route,
			Form:                 t.Form,
			MedicationID:         t.MedicationID,
			Dosage:               t.Dosage,
			DispenseType:         t.DispenseType,
			DispenseNumber:       uint32(t.DispenseNumber),
			Refills:              uint32(t.Refills),
			SubstitutionsAllowed: t.SubstitutionsAllowed,
			DaysSupply:           uint32(t.DaysSupply),
			Sig:                  t.Sig,
			PharmacyID:           t.PharmacyID,
			PharmacyInstructions: t.PharmacyInstructions,
		}
	}

	return cpr, nil
}

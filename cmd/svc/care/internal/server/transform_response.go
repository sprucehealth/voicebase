package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/care"
)

type answerModelToResponseTransformerFunc func(answer *models.Answer) (client.Answer, error)

var answerModelToResponseTransformers map[string]answerModelToResponseTransformerFunc

func init() {
	answerModelToResponseTransformers = map[string]answerModelToResponseTransformerFunc{
		"q_type_media_section":     transformMediaSectionToResponse,
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

func transformMediaSectionToResponse(answer *models.Answer) (client.Answer, error) {
	if answer.GetMediaSection() == nil {
		return nil, errors.Trace(fmt.Errorf("expected media section to be populated for answer but it wasnt"))
	}
	mediaSectionAnswer := &client.MediaQuestionAnswer{
		Type:     answer.Type,
		Sections: make([]*client.MediaSectionItem, len(answer.GetMediaSection().Sections)),
	}

	for i, mediaSection := range answer.GetMediaSection().Sections {
		mediaSectionAnswer.Sections[i] = &client.MediaSectionItem{
			Name:  mediaSection.Name,
			Slots: make([]*client.MediaSlotItem, len(mediaSection.Slots)),
		}

		for j, mediaSlot := range mediaSection.Slots {
			mediaSectionAnswer.Sections[i].Slots[j] = &client.MediaSlotItem{
				Name:    mediaSlot.Name,
				SlotID:  mediaSlot.SlotID,
				MediaID: mediaSlot.MediaID,
				URL:     "https://placekitten.com/600/800", //TODO
				Type:    "photo",                           //TODO : get from media service
			}
		}
	}
	return mediaSectionAnswer, nil
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

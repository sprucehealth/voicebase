package client

import (
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
)

func transformQuestion(question *saml.Question) (*layout.Question, error) {
	if question == nil {
		return nil, nil
	} else if question.Details == nil {
		return nil, errors.Trace(fmt.Errorf("expected details for question but got none"))
	}

	tQuestion := &layout.Question{
		ID:                 question.Details.Tag,
		Title:              question.Details.Text,
		TitleHasTokens:     tokenMatcher.Match([]byte(question.Details.Text)),
		Type:               question.Details.Type,
		Subtext:            question.Details.Subtext,
		Summary:            question.Details.Summary,
		AdditionalFields:   transformQuestionAdditionalFields(question.Details.AdditionalFields),
		PotentialAnswers:   transformAnswers(question.Details.Answers),
		Condition:          transformCondition(question.Condition),
		Required:           question.Details.Required,
		AlertFormattedText: question.Details.AlertText,
		ToAlert:            question.Details.ToAlert,
	}

	// TODO: Move the autocomplete params specification to the SAML layer.
	// However doing so currently requires a ton of updates to the SAML given that
	// allergy and medication questions are in almost every SAML. So for now
	// hardcoding the autocomplete params.
	if tQuestion.ID == "q_allergic_medication_entry" {
		if tQuestion.AdditionalFields == nil {
			tQuestion.AdditionalFields = &layout.QuestionAdditionalFields{}
		}
		tQuestion.AdditionalFields.AutocompleteParams = map[string]string{
			"source": "PATIENT_ALLERGY",
		}
	} else if tQuestion.ID == "q_current_medications_entry" {
		if tQuestion.AdditionalFields == nil {
			tQuestion.AdditionalFields = &layout.QuestionAdditionalFields{}
		}
		tQuestion.AdditionalFields.AutocompleteParams = map[string]string{
			"source": "PATIENT_DRUG",
		}
	}

	var err error
	tQuestion.PhotoSlots, err = transformPhotoSlots(question.Details.PhotoSlots)
	if err != nil {
		return nil, errors.Trace(err)
	}

	tQuestion.SubQuestionsConfig, err = transformSubquestionsConfig(question)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(question.Details.AnswerGroups) > 0 {
		tQuestion.PotentialAnswers = make([]*layout.PotentialAnswer, 0, 5*len(question.Details.AnswerGroups))
		if tQuestion.AdditionalFields == nil {
			tQuestion.AdditionalFields = &layout.QuestionAdditionalFields{}
		}
		tQuestion.AdditionalFields.AnswerGroups = make([]*layout.AnswerGroup, len(question.Details.AnswerGroups))

		for i, answerGroup := range question.Details.AnswerGroups {
			tQuestion.AdditionalFields.AnswerGroups[i] = &layout.AnswerGroup{
				Title: answerGroup.Title,
				Count: len(answerGroup.Answers),
			}
			tQuestion.PotentialAnswers = append(tQuestion.PotentialAnswers, transformAnswers(answerGroup.Answers)...)
		}
	}

	return tQuestion, nil
}

func transformQuestionAdditionalFields(additionalFields *saml.QuestionAdditionalFields) *layout.QuestionAdditionalFields {
	if additionalFields == nil {
		return nil
	}

	return &layout.QuestionAdditionalFields{
		PlaceholderText: additionalFields.PlaceholderText,
		Popup:           transformPopup(additionalFields.Popup),
		AllowsMultipleSections:  additionalFields.AllowsMultipleSections,
		UserDefinedSectionTitle: additionalFields.UserDefinedSectionTitle,
		AddButtonText:           additionalFields.AddButtonText,
		AddText:                 additionalFields.AddText,
		EmptyStateText:          additionalFields.EmptyStateText,
		RemoveButtonText:        additionalFields.RemoveButtonText,
		SaveButtonText:          additionalFields.SaveButtonText,
	}
}

func transformAnswers(answers []*saml.Answer) []*layout.PotentialAnswer {
	potentialAnswers := make([]*layout.PotentialAnswer, len(answers))
	for i, answer := range answers {
		potentialAnswers[i] = &layout.PotentialAnswer{
			ID:         answer.Tag,
			Answer:     answer.Text,
			Summary:    answer.Summary,
			Type:       answer.Type,
			ToAlert:    answer.ToAlert,
			ClientData: transformAnswerClientData(answer.ClientData),
		}
		if potentialAnswers[i].Summary == "" {
			potentialAnswers[i].Summary = potentialAnswers[i].Answer
		}
	}

	return potentialAnswers
}

func transformAnswerClientData(clientData *saml.AnswerClientData) *layout.AnswerClientData {
	if clientData == nil {
		return nil
	}

	return &layout.AnswerClientData{
		PlaceholderText: clientData.PlaceholderText,
		Popup:           transformPopup(clientData.Popup),
	}
}

func transformSubquestionsConfig(question *saml.Question) (*layout.SubQuestionsConfig, error) {
	if question.SubquestionConfig == nil {
		return nil, nil
	}

	tSubQuestionsConfig := &layout.SubQuestionsConfig{
		Screens: make([]*layout.Screen, len(question.SubquestionConfig.Screens)),
	}

	var err error
	for i, screen := range question.SubquestionConfig.Screens {
		tSubQuestionsConfig.Screens[i], err = transformScreen(screen)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return tSubQuestionsConfig, nil
}

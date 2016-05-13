package client

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/mapstructure"
)

// VisitAnswers is the object the client sends to set answers and clear some answers.
type VisitAnswers struct {
	// Answers is a map of questionID to answer to set for each question in the intake.
	Answers map[string]Answer `json:"answers,omitempty"`
	// ClearAnswers is a list of questionIDs to clear out answers for.
	ClearAnswers []string `json:"clear_answers,omitempty"`
}

func (v *VisitAnswers) DeleteNilAnswers() {
	for questionID, answer := range v.Answers {
		if answer == nil {
			delete(v.Answers, questionID)
		}
	}
}

// Answer is the client side representation of any answer to a question in an intake.
type Answer interface {
	mapstructure.Typed
	Validate(question *layout.Question) error
}

var typeRegistry *mapstructure.TypeRegistry = mapstructure.NewTypeRegistry()

func init() {
	typeRegistry.MustRegisterType(&PhotoQuestionAnswer{})
	typeRegistry.MustRegisterType(&MultipleChoiceQuestionAnswer{})
	typeRegistry.MustRegisterType(&FreeTextQuestionAnswer{})
	typeRegistry.MustRegisterType(&SingleSelectQuestionAnswer{})
	typeRegistry.MustRegisterType(&SegmentedControlQuestionAnswer{})
	typeRegistry.MustRegisterType(&SingleEntryQuestionAnswer{})
	typeRegistry.MustRegisterType(&AutocompleteQuestionAnswer{})
}

// PHOTO SECTION

type PhotoSlotItem struct {
	Name     string `json:"name"`
	SlotID   string `json:"slot_id"`
	PhotoID  string `json:"photo_id"`
	PhotoURL string `json:"url,omitempty"`
}

type PhotoSectionItem struct {
	Name  string           `json:"name"`
	Slots []*PhotoSlotItem `json:"photos"`
}

type PhotoQuestionAnswer struct {
	Type          string              `json:"type"`
	PhotoSections []*PhotoSectionItem `json:"photo_sections"`
}

func (p *PhotoQuestionAnswer) TypeName() string {
	return "q_type_photo_section"
}

func (p *PhotoQuestionAnswer) Validate(question *layout.Question) error {
	if err := validateQuestionType(p, question); err != nil {
		return errors.Trace(err)
	}
	if len(p.PhotoSections) == 0 {
		if isQuestionRequired(question) {
			return errors.Trace(fmt.Errorf("answer to question %s is required but empty list of photo sections submitted", question.ID))
		}
	}

	if question.AdditionalFields == nil || question.AdditionalFields.AllowsMultipleSections == nil || !*question.AdditionalFields.AllowsMultipleSections {
		if len(p.PhotoSections) > 1 {
			return errors.Trace(fmt.Errorf("answer to question %s can only have 1 photo section but has %d", question.ID, len(p.PhotoSections)))
		}
	}

	slotsFilled := make(map[string]struct{})
	for _, photoSection := range p.PhotoSections {
		if photoSection.Name == "" {
			return errors.Trace(fmt.Errorf("answer to question %s cannot have empty photo section name", question.ID))
		}
		if len(photoSection.Slots) == 0 {
			return errors.Trace(fmt.Errorf("answer to question %s cannot have a photo section with no photo slots", question.ID))
		}
		for _, photoSlot := range photoSection.Slots {
			if photoSlot.PhotoID == "" {
				return errors.Trace(fmt.Errorf("answer to question %s has a photo slot with no photo ID", question.ID))
			}
			if photoSlot.SlotID == "" {
				return errors.Trace(fmt.Errorf("answer to question %s has a photo slot with no slot ID", question.ID))
			}
			slotsFilled[photoSlot.SlotID] = struct{}{}

			// ensure that each slot is present in the question
			slotFound := false
			for _, slotInQuestion := range question.PhotoSlots {
				if slotInQuestion.ID == photoSlot.SlotID {
					slotFound = true
				}
			}
			if !slotFound {
				return errors.Trace(fmt.Errorf("answer to question %s has a photo slot referenced (%s) that does not exist in the question", question.ID, photoSlot.SlotID))
			}
		}
	}

	// ensure that there are no required photo slots in the question that did
	// not have an answer
	for _, photoSlot := range question.PhotoSlots {
		if _, ok := slotsFilled[photoSlot.ID]; !ok && photoSlot.Required != nil && *photoSlot.Required {
			return errors.Trace(fmt.Errorf("question %s has a required photo slot %s that is not answered", question.ID, photoSlot.ID))
		}
	}

	return nil
}

// MULTIPLE CHOICE

type PotentialAnswerItem struct {
	ID         string            `json:"id"`
	Text       string            `json:"text,omitempty"`
	Subanswers map[string]Answer `json:"answers,omitempty"`
}

func (p *PotentialAnswerItem) DeleteNilAnswers() {

	for questionID, answer := range p.Subanswers {
		if answer == nil {
			delete(p.Subanswers, questionID)
		}
	}

}

type MultipleChoiceQuestionAnswer struct {
	Type             string                 `json:"type"`
	PotentialAnswers []*PotentialAnswerItem `json:"potential_answers"`
}

func (m *MultipleChoiceQuestionAnswer) TypeName() string {
	return "q_type_multiple_choice"
}

func (m *MultipleChoiceQuestionAnswer) Validate(question *layout.Question) error {
	if err := validateQuestionType(m, question); err != nil {
		return errors.Trace(err)
	}

	if isQuestionRequired(question) {
		if len(m.PotentialAnswers) == 0 {
			return errors.Trace(fmt.Errorf("answer for question %s is required but empty list of potential answers specified", question.ID))
		}
	}

	// ensure that all potential answers exist in the question
	noneOfTheAboveSelected := false
	for _, optionSelected := range m.PotentialAnswers {
		optionFound := false
		for _, potentialAnswer := range question.PotentialAnswers {
			if potentialAnswer.ID == optionSelected.ID {
				if potentialAnswer.Type == "a_type_multiple_choice_none" {
					noneOfTheAboveSelected = true
				}
				if potentialAnswer.Type == "a_type_multiple_choice_other_free_text" {
					if optionSelected.Text == "" {
						return errors.Trace(fmt.Errorf("answer %s for question %s has a free text option selected but no free text specified", optionSelected.ID, question.ID))
					}
				} else if optionSelected.Text != "" {
					return errors.Trace(fmt.Errorf("answer %s for question %s has a free text response when one is not expected", optionSelected.ID, question.ID))
				}
				optionFound = true
			}
		}
		if !optionFound {
			return errors.Trace(fmt.Errorf("answer for question %s has a potential answer %s selected that is not found in the question", question.ID, optionSelected.ID))
		}

		if noneOfTheAboveSelected && len(m.PotentialAnswers) > 1 {
			return errors.Trace(fmt.Errorf("answer for question %s cannot have none of the above and another answer selected", question.ID))
		}

		optionSelected.DeleteNilAnswers()

		// validate subanswers
		if err := validateSubAnswers(optionSelected.ID, optionSelected.Subanswers, question); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// SINGLE SELECT

type SingleSelectQuestionAnswer struct {
	Type            string               `json:"type"`
	PotentialAnswer *PotentialAnswerItem `json:"potential_answer"`
}

func (s *SingleSelectQuestionAnswer) TypeName() string {
	return "q_type_single_select"
}

func (s *SingleSelectQuestionAnswer) Validate(question *layout.Question) error {
	if err := validateQuestionType(s, question); err != nil {
		return errors.Trace(err)
	}

	if isQuestionRequired(question) && s.PotentialAnswer == nil {
		return errors.Trace(fmt.Errorf("answer to question %s is required but no option selected", question.ID))
	}

	// ensure that the option selected exists in the question
	optionSelectedFound := false
	for _, potentialAnswer := range question.PotentialAnswers {
		if potentialAnswer.ID == s.PotentialAnswer.ID {
			optionSelectedFound = true
		}
	}
	if !optionSelectedFound {
		return errors.Trace(fmt.Errorf("answer to question %s has an option selected %s that does not exist", question.ID, s.PotentialAnswer.ID))
	}

	return nil
}

// FREE TEXT

type FreeTextQuestionAnswer struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (f *FreeTextQuestionAnswer) TypeName() string {
	return "q_type_free_text"
}

func (f *FreeTextQuestionAnswer) Validate(question *layout.Question) error {
	if err := validateQuestionType(f, question); err != nil {
		return errors.Trace(err)
	}

	if isQuestionRequired(question) && strings.TrimSpace(f.Text) == "" {
		return errors.Trace(fmt.Errorf("question %s is required but no text specified", question.ID))
	}
	return nil
}

// SEGMENTED CONTROL

type SegmentedControlQuestionAnswer struct {
	Type            string               `json:"type"`
	PotentialAnswer *PotentialAnswerItem `json:"potential_answer"`
}

func (s *SegmentedControlQuestionAnswer) TypeName() string {
	return "q_type_segmented_control"
}

func (s *SegmentedControlQuestionAnswer) Validate(question *layout.Question) error {
	if err := validateQuestionType(s, question); err != nil {
		return errors.Trace(err)
	}

	if isQuestionRequired(question) && s.PotentialAnswer == nil {
		return errors.Trace(fmt.Errorf("answer to question %s is required but no option selected", question.ID))
	}

	// ensure that the option selected exists in the question
	optionSelectedFound := false
	for _, potentialAnswer := range question.PotentialAnswers {
		if potentialAnswer.ID == s.PotentialAnswer.ID {
			optionSelectedFound = true
		}
	}
	if !optionSelectedFound {
		return errors.Trace(fmt.Errorf("answer to question %s has an option selected %s that does not exist", question.ID, s.PotentialAnswer.ID))
	}

	return nil
}

// SINGLE ENTRY

type SingleEntryQuestionAnswer struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *SingleEntryQuestionAnswer) TypeName() string {
	return "q_type_single_entry"
}

func (s *SingleEntryQuestionAnswer) Validate(question *layout.Question) error {
	if err := validateQuestionType(s, question); err != nil {
		return errors.Trace(err)
	}

	if isQuestionRequired(question) && strings.TrimSpace(s.Text) == "" {
		return errors.Trace(fmt.Errorf("question %s is required but no text specified", question.ID))
	}
	return nil
}

// AUTO COMPLETE

type AutocompleteItem struct {
	Text       string            `json:"text"`
	Subanswers map[string]Answer `json:"answers,omitempty"`
}

func (a *AutocompleteItem) DeleteNilAnswers() {

	for questionID, answer := range a.Subanswers {
		if answer == nil {
			delete(a.Subanswers, questionID)
		}
	}
}

type AutocompleteQuestionAnswer struct {
	Type    string              `json:"type"`
	Answers []*AutocompleteItem `json:"items"`
}

func (a *AutocompleteQuestionAnswer) TypeName() string {
	return "q_type_autocomplete"
}

func (a *AutocompleteQuestionAnswer) Validate(question *layout.Question) error {
	if err := validateQuestionType(a, question); err != nil {
		return errors.Trace(err)
	}

	if isQuestionRequired(question) && len(a.Answers) == 0 {
		return errors.Trace(fmt.Errorf("question %s is required but no answer specified", question.ID))
	}

	for _, item := range a.Answers {
		if item.Text == "" {
			return errors.Trace(fmt.Errorf("question %s has an empty answer entry", question.ID))
		}
		if err := validateSubAnswers(item.Text, item.Subanswers, question); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func isQuestionRequired(question *layout.Question) bool {
	return question.Required != nil && *question.Required
}

func validateQuestionType(answer Answer, question *layout.Question) error {
	if answer.TypeName() != question.Type {
		return fmt.Errorf("answer of type %s does not match question type %s", answer.TypeName(), question.Type)
	}
	return nil
}

func validateSubAnswers(optionSelectedID string, subanswers map[string]Answer, question *layout.Question) error {

	subQuestionsAnswered := make(map[string]struct{})
	for subQuestionID, subanswer := range subanswers {

		// ensure that the question exists in the subquestion config
		if question.SubQuestionsConfig == nil {
			return fmt.Errorf("question %s doed not have a subquestion config but subanswers specified for it", question.ID)
		}

		subQuestionFound := false
		for _, subScreen := range question.SubQuestionsConfig.Screens {
			for _, subQuestion := range subScreen.Questions {
				if subQuestion.ID == subQuestionID {
					subQuestionFound = true
					if err := subanswer.Validate(subQuestion); err != nil {
						return fmt.Errorf("answer %s for question %s has an invalid subanswer: %s", optionSelectedID, question.ID, err)
					}
					break
				}
			}
		}
		if !subQuestionFound {
			return fmt.Errorf("answer %s for question %s has a subanswer for which the subquestion %s was not found", optionSelectedID, question.ID, subQuestionID)
		}
		subQuestionsAnswered[subQuestionID] = struct{}{}
	}
	return nil
}

package manager

import "encoding/json"

func init() {
	mustRegisterAnswer(questionTypeFreeText.String(), &freeTextAnswer{})
	mustRegisterAnswer(questionTypeSingleEntry.String(), &freeTextAnswer{})
	mustRegisterAnswer(questionTypeAutocomplete.String(), &autocompleteAnswer{})
	mustRegisterAnswer(questionTypeMultipleChoice.String(), &multipleChoiceAnswer{})
	mustRegisterAnswer(questionTypeSingleSelect.String(), &multipleChoiceAnswer{})
	mustRegisterAnswer(questionTypeSegmentedControl.String(), &multipleChoiceAnswer{})
	mustRegisterAnswer(questionTypePhoto.String(), &photoSectionAnswer{})
}

// textAnswerClientJSONItem represents a single answer selection/entry
// to be recorded as part of a textAnswerClientJSON object
type textAnswerClientJSONItem struct {
	PotentialAnswerID string            `json:"potential_answer_id,omitempty"`
	Text              string            `json:"answer_text,omitempty"`
	Items             []json.RawMessage `json:"answers,omitempty"`
}

// textAnswerClientJSON represents the structure used to communicate any text
// based patient answers (multiple choice, single select, free text, etc.).
// Unfortunately, the exact same representation is used for all text based answers
// which makes the representation of the answer generic versus specific to the question
// being answered.
type textAnswerClientJSON struct {
	QuestionID string                      `json:"question_id"`
	Items      []*textAnswerClientJSONItem `json:"potential_answers"`
}

func emptyTextAnswer(questionID string) ([]byte, error) {
	return json.Marshal(textAnswerClientJSON{
		QuestionID: questionID,
		Items:      []*textAnswerClientJSONItem{},
	})
}

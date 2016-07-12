package manager

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type answerItem struct {
	Text       string                   `json:"text"`
	SubAnswers map[string]patientAnswer `json:"answers,omitempty"`

	subScreens []screen
}

func (a *answerItem) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("autocomplete_answer_item", "text"); err != nil {
		return errors.Trace(err)
	}

	a.Text = data.mustGetString("text")

	subanswers := data.get("answers")
	if subanswers == nil {
		return nil
	}

	subanswersMap, err := getDataMap(subanswers)
	if err != nil {
		return errors.Trace(err)
	}

	a.SubAnswers = make(map[string]patientAnswer, len(subanswersMap))
	for questionID, subanswer := range subanswersMap {
		subAnswerMap, err := getDataMap(subanswer)
		if err != nil {
			return errors.Trace(err)
		}

		a.SubAnswers[questionID], err = getPatientAnswer(subAnswerMap)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (a *answerItem) text() string {
	return a.Text
}

func (a *answerItem) potentialAnswerID() string {
	return ""
}

func (a *answerItem) subAnswers() map[string]patientAnswer {
	return a.SubAnswers
}

func (a *answerItem) setSubscreens(screens []screen) {
	a.subScreens = screens
}

func (a *answerItem) subscreens() []screen {
	return a.subScreens
}

type autocompleteAnswer struct {
	Answers []topLevelAnswerItem `json:"items"`
}

func (a *autocompleteAnswer) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	for _, aItem := range a.Answers {
		b.WriteString("\n")
		b.WriteString(indentAtDepth(indent, depth) + "A: " + aItem.text())
		for _, ssItem := range aItem.subscreens() {
			b.WriteString(indentAtDepth(indent, depth) + ssItem.stringIndent(indent, depth))
		}
	}

	return b.String()
}

func (a *autocompleteAnswer) topLevelAnswers() []topLevelAnswerItem {
	return a.Answers
}

func (a *autocompleteAnswer) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("autocomplete_answer", "items"); err != nil {
		return err
	}

	answers, err := data.getInterfaceSlice("items")
	if err != nil {
		return err
	}

	a.Answers = make([]topLevelAnswerItem, len(answers))
	for i, aItem := range answers {
		answerData, err := getDataMap(aItem)
		if err != nil {
			return err
		}

		item := &answerItem{}
		a.Answers[i] = item
		if err := item.unmarshalMapFromClient(answerData); err != nil {
			return err
		}
	}
	return nil
}

func (a *autocompleteAnswer) unmarshalProtobuf(data []byte) error {
	var pb intake.AutocompletePatientAnswer
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}

	a.Answers = make([]topLevelAnswerItem, len(pb.Answers))
	for i, answerText := range pb.Answers {
		a.Answers[i] = &answerItem{
			Text: answerText,
		}
	}

	return nil
}

func (a *autocompleteAnswer) transformToProtobuf() (proto.Message, error) {
	var pb intake.AutocompletePatientAnswer
	pb.Answers = make([]string, len(a.Answers))
	for i, aItem := range a.Answers {
		pb.Answers[i] = aItem.text()
	}

	return &pb, nil
}

type autocompleteAnswerClientJSON struct {
	Type  string                              `json:"type"`
	Items []*autocompleteAnswerItemClientJSON `json:"items"`
}

type autocompleteAnswerItemClientJSON struct {
	Text       string                 `json:"text"`
	Subanswers map[string]interface{} `json:"answers,omitempty"`
}

func (a *autocompleteAnswer) transformForClient() (interface{}, error) {
	clientJSON := &autocompleteAnswerClientJSON{
		Type:  questionTypeAutocomplete.String(),
		Items: make([]*autocompleteAnswerItemClientJSON, len(a.Answers)),
	}

	for i, item := range a.Answers {
		aItem := item.(*answerItem)

		clientJSON.Items[i] = &autocompleteAnswerItemClientJSON{
			Text:       aItem.text(),
			Subanswers: make(map[string]interface{}, len(aItem.SubAnswers)),
		}

		// add answers for all visible questions within subscreens
		// as subanswers
		for _, subscreenItem := range aItem.subScreens {
			qContainer, ok := subscreenItem.(questionsContainer)
			if ok {
				for _, sqItem := range qContainer.questions() {

					_, err := sqItem.patientAnswer()
					if err != nil {
						continue
					}

					subanswerData, err := sqItem.answerForClient()
					if err != nil {
						return nil, err
					}

					clientJSON.Items[i].Subanswers[sanitizeQuestionID(sqItem.id())] = subanswerData
				}
			}
		}
	}

	return clientJSON, nil
}

func (a *autocompleteAnswer) equals(other patientAnswer) bool {
	if a == nil && other == nil {
		return true
	} else if a == nil || other == nil {
		return false
	}

	otherAA, ok := other.(*autocompleteAnswer)
	if !ok {
		return false
	}

	if len(a.Answers) != len(otherAA.Answers) {
		return false
	}

	for i, aItem := range a.Answers {
		item := aItem.(*answerItem)
		otherItem := otherAA.Answers[i].(*answerItem)
		if item.Text != otherItem.Text {
			return false
		}

		// Note: Intentionally not including subscreens in the equality check since
		// the answer for a question is set piece-wise where we do an equality check for
		// the immediate answer to the question versus the answer as a whole.
	}

	return true
}

func (a *autocompleteAnswer) isEmpty() bool {
	return len(a.Answers) == 0
}

package manager

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type answerItem struct {
	Text       string          `json:"text"`
	SubAnswers []patientAnswer `json:"sub_answers,omitempty"`
	subScreens []screen
}

func (a *answerItem) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("autocomplete_answer_item", "answer_text", "type"); err != nil {
		return err
	}

	a.Text = data.mustGetString("answer_text")
	subanswers, err := data.getInterfaceSlice("answers")
	if err != nil {
		return err
	}

	a.SubAnswers = make([]patientAnswer, len(subanswers))
	for i, subanswer := range subanswers {
		subAnswerMap, err := getDataMap(subanswer)
		if err != nil {
			return err
		}

		a.SubAnswers[i], err = getPatientAnswer(subAnswerMap)
		if err != nil {
			return err
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

func (a *answerItem) subAnswers() []patientAnswer {
	return a.SubAnswers
}

func (a *answerItem) setSubscreens(screens []screen) {
	a.subScreens = screens
}

func (a *answerItem) subscreens() []screen {
	return a.subScreens
}

type autocompleteAnswer struct {
	QuestionID string               `json:"question_id"`
	Answers    []topLevelAnswerItem `json:"answers"`
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

func (a *autocompleteAnswer) setQuestionID(questionID string) {
	a.QuestionID = questionID
}

func (a *autocompleteAnswer) questionID() string {
	return a.QuestionID
}

func (a *autocompleteAnswer) topLevelAnswers() []topLevelAnswerItem {
	return a.Answers
}

func (a *autocompleteAnswer) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("autocomplete_answer", "answers"); err != nil {
		return err
	}

	answers, err := data.getInterfaceSlice("answers")
	if err != nil {
		return err
	}

	var questionID string
	a.Answers = make([]topLevelAnswerItem, len(answers))
	for i, aItem := range answers {
		answerData, err := getDataMap(aItem)
		if err != nil {
			return err
		}
		questionIDForItem := answerData.mustGetString("question_id")
		if questionID == "" {
			questionID = questionIDForItem
		} else if questionID != questionIDForItem {
			return errors.New("question_id in each answer item doesn't match")
		}

		item := &answerItem{}
		a.Answers[i] = item
		if err := item.unmarshalMapFromClient(answerData); err != nil {
			return err
		}
	}
	a.QuestionID = questionID
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

func (a *autocompleteAnswer) marshalEmptyJSONForClient() ([]byte, error) {
	return emptyTextAnswer(sanitizeQuestionID(a.QuestionID))
}

func (a *autocompleteAnswer) marshalJSONForClient() ([]byte, error) {
	clientJSON := textAnswerClientJSON{
		QuestionID: sanitizeQuestionID(a.QuestionID),
		Items:      make([]*textAnswerClientJSONItem, len(a.Answers)),
	}

	for i, item := range a.Answers {
		aItem := item.(*answerItem)

		clientJSON.Items[i] = &textAnswerClientJSONItem{
			Text:  aItem.text(),
			Items: make([]json.RawMessage, len(aItem.SubAnswers)),
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

					subanswerData, err := sqItem.marshalAnswerForClient()
					if err != nil {
						return nil, err
					}

					clientJSON.Items[i].Items = append(clientJSON.Items[i].Items, json.RawMessage(subanswerData))
				}
			}
		}
	}

	return json.Marshal(clientJSON)
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

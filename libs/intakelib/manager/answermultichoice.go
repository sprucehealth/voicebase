package manager

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type multipleChoiceAnswerSelection struct {
	Text                string          `json:"text,omitempty"`
	PotentialAnswerText string          `json:"-"`
	PotentialAnswerID   string          `json:"potential_answer_id,omitempty"`
	SubAnswers          []patientAnswer `json:"sub_answers"`

	subScreens []screen
}

func (m *multipleChoiceAnswerSelection) unmarshalMapFromClient(data dataMap) error {
	m.PotentialAnswerID = data.mustGetString("potential_answer_id")
	m.Text = data.mustGetString("answer_text")
	m.PotentialAnswerText = data.mustGetString("potential_answer")

	subanswers, err := data.getInterfaceSlice("answers")
	if err != nil {
		return err
	}

	m.SubAnswers = make([]patientAnswer, len(subanswers))
	for i, subanswer := range subanswers {
		subAnswerMap, err := getDataMap(subanswer)
		if err != nil {
			return err
		}

		m.SubAnswers[i], err = getPatientAnswer(subAnswerMap)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *multipleChoiceAnswerSelection) text() string {
	if m.Text != "" {
		return m.Text
	}
	return m.PotentialAnswerText
}

func (m *multipleChoiceAnswerSelection) potentialAnswerID() string {
	return m.PotentialAnswerID
}

func (m *multipleChoiceAnswerSelection) subAnswers() []patientAnswer {
	return m.SubAnswers
}

func (m *multipleChoiceAnswerSelection) setSubscreens(screens []screen) {
	m.subScreens = screens
}

func (m *multipleChoiceAnswerSelection) subscreens() []screen {
	return m.subScreens
}

type multipleChoiceAnswer struct {
	Answers    []topLevelAnswerItem `json:"answers"`
	QuestionID string               `json:"question_id"`
}

func (a *multipleChoiceAnswer) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	for _, aItem := range a.Answers {
		b.WriteString("\n")
		b.WriteString(indentAtDepth(indent, depth) + "A: " + aItem.text())

		for _, ssItem := range aItem.subscreens() {
			b.WriteString("\n")
			b.WriteString(ssItem.stringIndent(indent, depth+1))
		}
	}

	return b.String()
}

func (m *multipleChoiceAnswer) setQuestionID(questionID string) {
	m.QuestionID = questionID
}

func (m *multipleChoiceAnswer) questionID() string {
	return m.QuestionID
}

func (m *multipleChoiceAnswer) topLevelAnswers() []topLevelAnswerItem {
	return m.Answers
}

func (m *multipleChoiceAnswer) unmarshalMapFromClient(data dataMap) error {

	// top level multiple choice answers are represented as an array of answer
	// selections/entries, while multiple choice answers for subquestions are represented
	// as a single object (with the current limitation being that multiple selections
	// for a single subquestion is not supported due to the current structure).
	answers, err := data.getInterfaceSlice("answers")
	if err != nil {
		return err
	} else if len(answers) == 0 {
		answers = []interface{}{data}
	}

	var questionID string
	m.Answers = make([]topLevelAnswerItem, len(answers))
	for i, selectionItem := range answers {

		answerMap, err := getDataMap(selectionItem)
		if err != nil {
			return err
		}

		questionIDFromItem := answerMap.mustGetString("question_id")
		if questionID == "" {
			questionID = questionIDFromItem
		} else if questionID != questionIDFromItem {
			return fmt.Errorf("question_id mismatch between answer items")
		}

		selection := &multipleChoiceAnswerSelection{}
		m.Answers[i] = selection
		if err := selection.unmarshalMapFromClient(answerMap); err != nil {
			return err
		}
	}
	m.QuestionID = questionID
	return nil
}

func (m *multipleChoiceAnswer) unmarshalProtobuf(data []byte) error {
	var pb intake.MultipleChoicePatientAnswer
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}

	m.Answers = make([]topLevelAnswerItem, len(pb.AnswerSelections))
	for i, selectionItem := range pb.AnswerSelections {
		s := &multipleChoiceAnswerSelection{}
		if selectionItem.Text != nil {
			s.Text = *selectionItem.Text
		}
		if selectionItem.PotentialAnswerId != nil {
			s.PotentialAnswerID = *selectionItem.PotentialAnswerId
		}
		m.Answers[i] = s
	}

	return nil
}

func (m *multipleChoiceAnswer) transformToProtobuf() (proto.Message, error) {
	var pb intake.MultipleChoicePatientAnswer
	pb.AnswerSelections = make([]*intake.MultipleChoicePatientAnswer_Selection, len(m.Answers))
	for i, selectionItem := range m.Answers {
		selection := selectionItem.(*multipleChoiceAnswerSelection)

		s := &intake.MultipleChoicePatientAnswer_Selection{
			Text: proto.String(selection.Text),
		}
		if selection.PotentialAnswerID != "" {
			s.PotentialAnswerId = proto.String(selection.PotentialAnswerID)
		}

		pb.AnswerSelections[i] = s
	}
	return &pb, nil
}

func (m *multipleChoiceAnswer) marshalEmptyJSONForClient() ([]byte, error) {
	return emptyTextAnswer(sanitizeQuestionID(m.QuestionID))
}

func (m *multipleChoiceAnswer) marshalJSONForClient() ([]byte, error) {
	clientJSON := &textAnswerClientJSON{
		QuestionID: sanitizeQuestionID(m.QuestionID),
		Items:      make([]*textAnswerClientJSONItem, len(m.Answers)),
	}

	for i, aItem := range m.Answers {
		selection := aItem.(*multipleChoiceAnswerSelection)
		clientJSON.Items[i] = &textAnswerClientJSONItem{
			Text:              selection.Text,
			PotentialAnswerID: selection.PotentialAnswerID,
		}

		// add answers for all visible questions within subscreens
		// as subanswers
		for _, subscreenItem := range selection.subScreens {
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

func (m *multipleChoiceAnswer) equals(other patientAnswer) bool {
	if m == nil && other == nil {
		return true
	} else if m == nil || other == nil {
		return false
	}

	otherMCA, ok := other.(*multipleChoiceAnswer)
	if !ok {
		return false
	}

	if len(m.Answers) != len(otherMCA.Answers) {
		return false
	}

	for i, aItem := range m.Answers {
		sel := aItem.(*multipleChoiceAnswerSelection)
		otherSel := otherMCA.Answers[i].(*multipleChoiceAnswerSelection)
		if sel.PotentialAnswerID != otherSel.PotentialAnswerID {
			return false
		} else if sel.Text != otherSel.Text {
			return false
		}

		// Note: Intentionally not including subscreens in the equality check since
		// the answer for a question is set piece-wise where we do an equality check for
		// the immediate answer to the question versus the answer as a whole.
	}

	return true
}

func (m *multipleChoiceAnswer) isEmpty() bool {
	return len(m.Answers) == 0
}

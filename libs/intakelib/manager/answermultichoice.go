package manager

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type multipleChoiceAnswerSelection struct {
	Text                string                   `json:"text,omitempty"`
	PotentialAnswerText string                   `json:"-"`
	PotentialAnswerID   string                   `json:"id,omitempty"`
	SubAnswers          map[string]patientAnswer `json:"answers"`

	subScreens []screen
}

func (m *multipleChoiceAnswerSelection) unmarshalMapFromClient(data dataMap) error {
	m.PotentialAnswerID = data.mustGetString("id")
	m.Text = data.mustGetString("text")

	subanswers := data.get("answers")
	if subanswers == nil {
		return nil
	}

	subanswersMap, err := getDataMap(data.get("answers"))
	if err != nil {
		return errors.Trace(err)
	}

	m.SubAnswers = make(map[string]patientAnswer, len(subanswersMap))
	for questionID, subanswer := range subanswersMap {
		subAnswerMap, err := getDataMap(subanswer)
		if err != nil {
			return errors.Trace(err)
		}

		m.SubAnswers[questionID], err = getPatientAnswer(subAnswerMap)
		if err != nil {
			return errors.Trace(err)
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

func (m *multipleChoiceAnswerSelection) subAnswers() map[string]patientAnswer {
	return m.SubAnswers
}

func (m *multipleChoiceAnswerSelection) setSubscreens(screens []screen) {
	m.subScreens = screens
}

func (m *multipleChoiceAnswerSelection) subscreens() []screen {
	return m.subScreens
}

type multipleChoiceAnswer struct {
	Answers []topLevelAnswerItem `json:"potential_answers"`
}

func (a *multipleChoiceAnswer) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	for _, aItem := range a.Answers {
		b.WriteString("\n")
		text := aItem.text()
		if text == "" {
			text = aItem.potentialAnswerID()
		}
		b.WriteString(indentAtDepth(indent, depth) + "A: " + text)

		for _, ssItem := range aItem.subscreens() {
			b.WriteString("\n")
			b.WriteString(ssItem.stringIndent(indent, depth+1))
		}
	}

	return b.String()
}

func (m *multipleChoiceAnswer) topLevelAnswers() []topLevelAnswerItem {
	return m.Answers
}

func (m *multipleChoiceAnswer) unmarshalMapFromClient(data dataMap) error {

	answers, err := data.getInterfaceSlice("potential_answers")
	if err != nil {
		return err
	} else if len(answers) == 0 {
		answers = []interface{}{data}
	}

	m.Answers = make([]topLevelAnswerItem, len(answers))
	for i, selectionItem := range answers {

		answerMap, err := getDataMap(selectionItem)
		if err != nil {
			return err
		}

		selection := &multipleChoiceAnswerSelection{}
		m.Answers[i] = selection
		if err := selection.unmarshalMapFromClient(answerMap); err != nil {
			return err
		}
	}
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

type multipleChoiceAnswerItemClientJSON struct {
	ID         string                 `json:"id"`
	Text       string                 `json:"text"`
	Subanswers map[string]interface{} `json:"answers,omitempty"`
}

type multipleChoiceAnswerClientJSON struct {
	Type             string                                `json:"type"`
	PotentialAnswers []*multipleChoiceAnswerItemClientJSON `json:"potential_answers"`
}

func (m *multipleChoiceAnswer) transformForClient() (interface{}, error) {
	clientJSON := &multipleChoiceAnswerClientJSON{
		Type:             questionTypeMultipleChoice.String(),
		PotentialAnswers: make([]*multipleChoiceAnswerItemClientJSON, len(m.Answers)),
	}

	for i, aItem := range m.Answers {
		selection := aItem.(*multipleChoiceAnswerSelection)
		clientJSON.PotentialAnswers[i] = &multipleChoiceAnswerItemClientJSON{
			Text:       selection.Text,
			ID:         selection.PotentialAnswerID,
			Subanswers: make(map[string]interface{}),
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

					subanswerData, err := sqItem.answerForClient()
					if err != nil {
						return nil, err
					}

					clientJSON.PotentialAnswers[i].Subanswers[sqItem.id()] = subanswerData
				}
			}
		}
	}

	return clientJSON, nil
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

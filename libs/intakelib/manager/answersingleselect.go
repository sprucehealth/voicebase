package manager

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
	"github.com/sprucehealth/backend/libs/ptr"
)

type singleSelectAnswer struct {
	Answer topLevelAnswerItem `json:"potential_answer"`
}

func (a *singleSelectAnswer) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	b.WriteString("\n")
	text := a.Answer.text()
	if text == "" {
		text = a.Answer.potentialAnswerID()
	}
	b.WriteString(indentAtDepth(indent, depth) + "A: " + text)

	for _, ssItem := range a.Answer.subscreens() {
		b.WriteString("\n")
		b.WriteString(ssItem.stringIndent(indent, depth+1))
	}

	return b.String()
}

func (m *singleSelectAnswer) topLevelAnswers() []topLevelAnswerItem {
	return []topLevelAnswerItem{m.Answer}
}

func (m *singleSelectAnswer) unmarshalMapFromClient(data dataMap) error {

	answerMap, err := getDataMap(data.get("potential_answer"))
	if err != nil {
		return err
	}

	selection := &multipleChoiceAnswerSelection{}
	m.Answer = selection
	if err := selection.unmarshalMapFromClient(answerMap); err != nil {
		return err
	}
	return nil
}

func (m *singleSelectAnswer) unmarshalProtobuf(data []byte) error {
	var pb intake.SingleSelectPatientAnswer
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}

	a := &multipleChoiceAnswerSelection{}
	if pb.Text != nil {
		a.Text = *pb.Text
	}
	if pb.PotentialAnswerId != nil {
		a.PotentialAnswerID = *pb.PotentialAnswerId
	}
	m.Answer = a

	return nil
}

func (m *singleSelectAnswer) transformToProtobuf() (proto.Message, error) {

	a := m.Answer.(*multipleChoiceAnswerSelection)
	pb := &intake.SingleSelectPatientAnswer{
		Text:              proto.String(a.Text),
		PotentialAnswerId: ptr.String(a.PotentialAnswerID),
	}

	return pb, nil
}

type singleSelectAnswerClientJSON struct {
	Type            string                              `json:"type"`
	PotentialAnswer *multipleChoiceAnswerItemClientJSON `json:"potential_answer"`
}

func (m *singleSelectAnswer) transformForClient() (interface{}, error) {
	a := m.Answer.(*multipleChoiceAnswerSelection)
	clientJSON := &singleSelectAnswerClientJSON{
		Type: questionTypeSingleSelect.String(),
		PotentialAnswer: &multipleChoiceAnswerItemClientJSON{
			Text: a.Text,
			ID:   a.PotentialAnswerID,
		},
	}

	return clientJSON, nil
}

func (m *singleSelectAnswer) equals(other patientAnswer) bool {
	if m == nil && other == nil {
		return true
	} else if m == nil || other == nil {
		return false
	}

	otherSSA, ok := other.(*singleSelectAnswer)
	if !ok {
		return false
	}

	if m.Answer == nil || otherSSA.Answer == nil {
		return false
	}

	sel := m.Answer.(*multipleChoiceAnswerSelection)
	otherSel := otherSSA.Answer.(*multipleChoiceAnswerSelection)
	if sel.PotentialAnswerID != otherSel.PotentialAnswerID {
		return false
	} else if sel.Text != otherSel.Text {
		return false
	}
	return true
}

func (m *singleSelectAnswer) isEmpty() bool {
	return m.Answer == nil
}

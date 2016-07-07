package manager

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
	"github.com/sprucehealth/backend/libs/ptr"
)

type segmentedControlAnswer struct {
	Answer topLevelAnswerItem `json:"potential_answer"`
}

func (a *segmentedControlAnswer) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	b.WriteString("\n")
	b.WriteString(indentAtDepth(indent, depth) + "A: " + a.Answer.text())

	for _, ssItem := range a.Answer.subscreens() {
		b.WriteString("\n")
		b.WriteString(ssItem.stringIndent(indent, depth+1))
	}

	return b.String()
}

func (m *segmentedControlAnswer) topLevelAnswers() []topLevelAnswerItem {
	return []topLevelAnswerItem{m.Answer}
}

func (m *segmentedControlAnswer) unmarshalMapFromClient(data dataMap) error {

	answerMap, err := getDataMap(data.get("potential_answer"))
	if err != nil {
		return errors.Trace(err)
	}

	selection := &multipleChoiceAnswerSelection{}
	m.Answer = selection
	if err := selection.unmarshalMapFromClient(answerMap); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (m *segmentedControlAnswer) unmarshalProtobuf(data []byte) error {
	var pb intake.SegmentedControlPatientAnswer
	if err := proto.Unmarshal(data, &pb); err != nil {
		return errors.Trace(err)
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

func (m *segmentedControlAnswer) transformToProtobuf() (proto.Message, error) {

	a := m.Answer.(*multipleChoiceAnswerSelection)
	pb := &intake.SegmentedControlPatientAnswer{
		Text:              proto.String(a.Text),
		PotentialAnswerId: ptr.String(a.PotentialAnswerID),
	}

	return pb, nil
}

type segmentedControlAnswerClientJSON struct {
	Type            string                              `json:"type"`
	PotentialAnswer *multipleChoiceAnswerItemClientJSON `json:"potential_answer"`
}

func (m *segmentedControlAnswer) transformForClient() (interface{}, error) {
	a := m.Answer.(*multipleChoiceAnswerSelection)
	clientJSON := &segmentedControlAnswerClientJSON{
		Type: questionTypeSegmentedControl.String(),
		PotentialAnswer: &multipleChoiceAnswerItemClientJSON{
			Text: a.Text,
			ID:   a.PotentialAnswerID,
		},
	}

	return clientJSON, nil
}

func (m *segmentedControlAnswer) equals(other patientAnswer) bool {
	if m == nil && other == nil {
		return true
	} else if m == nil || other == nil {
		return false
	}

	otherSCA, ok := other.(*segmentedControlAnswer)
	if !ok {
		return false
	}

	if m.Answer == nil || otherSCA.Answer == nil {
		return false
	}

	sel := m.Answer.(*multipleChoiceAnswerSelection)
	otherSel := otherSCA.Answer.(*multipleChoiceAnswerSelection)
	if sel.PotentialAnswerID != otherSel.PotentialAnswerID {
		return false
	} else if sel.Text != otherSel.Text {
		return false
	}
	return true
}

func (m *segmentedControlAnswer) isEmpty() bool {
	return m.Answer == nil
}

package manager

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type singleEntryAnswer struct {
	Text string `json:"text,omitempty"`
}

func (f *singleEntryAnswer) stringIndent(indent string, depth int) string {
	return fmt.Sprintf("\n"+indentAtDepth(indent, depth)+"A: %s", f.Text)
}

func (f *singleEntryAnswer) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("single_entry_answer", "text"); err != nil {
		return err
	}

	f.Text = data.mustGetString("text")

	return nil
}

func (f *singleEntryAnswer) unmarshalProtobuf(data []byte) error {
	var pb intake.SingleEntryPatientAnswer
	if err := proto.Unmarshal(data, &pb); err != nil {
		return nil
	}

	if pb.Text != nil {
		f.Text = *pb.Text
	}
	return nil
}

func (f *singleEntryAnswer) transformToProtobuf() (proto.Message, error) {
	return &intake.SingleEntryPatientAnswer{
		Text: proto.String(f.Text),
	}, nil
}

type singleEntryAnswerClientJSON struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (f *singleEntryAnswer) transformForClient() (interface{}, error) {
	clientJSON := &singleEntryAnswerClientJSON{
		Type: questionTypeSingleEntry.String(),
		Text: f.Text,
	}

	return clientJSON, nil
}

func (f *singleEntryAnswer) equals(other patientAnswer) bool {
	if f == nil && other == nil {
		return true
	} else if f == nil || other == nil {
		return false
	}

	otherSEA, ok := other.(*singleEntryAnswer)
	if !ok {
		return false
	}

	return f.Text == otherSEA.Text
}

func (f *singleEntryAnswer) isEmpty() bool {
	return len(f.Text) == 0 || len(strings.TrimSpace(f.Text)) == 0
}

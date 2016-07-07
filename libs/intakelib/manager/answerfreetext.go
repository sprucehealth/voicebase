package manager

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type freeTextAnswer struct {
	Text string `json:"text,omitempty"`
}

func (f *freeTextAnswer) stringIndent(indent string, depth int) string {
	return fmt.Sprintf("\n"+indentAtDepth(indent, depth)+"A: %s", f.Text)
}

func (f *freeTextAnswer) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("free_text_answer", "text"); err != nil {
		return errors.Trace(err)
	}

	f.Text = data.mustGetString("text")

	return nil
}

func (f *freeTextAnswer) unmarshalProtobuf(data []byte) error {
	var pb intake.FreeTextPatientAnswer
	if err := proto.Unmarshal(data, &pb); err != nil {
		return nil
	}

	if pb.Text != nil {
		f.Text = *pb.Text
	}
	return nil
}

func (f *freeTextAnswer) transformToProtobuf() (proto.Message, error) {
	return &intake.FreeTextPatientAnswer{
		Text: proto.String(f.Text),
	}, nil
}

type freeTextAnswerClientJSON struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (f *freeTextAnswer) transformForClient() (interface{}, error) {
	clientJSON := &freeTextAnswerClientJSON{
		Type: questionTypeFreeText.String(),
		Text: f.Text,
	}

	return clientJSON, nil
}

func (f *freeTextAnswer) equals(other patientAnswer) bool {
	if f == nil && other == nil {
		return true
	} else if f == nil || other == nil {
		return false
	}

	otherFTA, ok := other.(*freeTextAnswer)
	if !ok {
		return false
	}

	return f.Text == otherFTA.Text
}

func (f *freeTextAnswer) isEmpty() bool {
	return len(f.Text) == 0 || len(strings.TrimSpace(f.Text)) == 0
}

package manager

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type freeTextAnswer struct {
	QuestionID string `json:"question_id,string"`
	Text       string `json:"text,omitempty"`
}

func (f *freeTextAnswer) stringIndent(indent string, depth int) string {
	return fmt.Sprintf("\n"+indentAtDepth(indent, depth)+"A: %s", f.Text)
}

func (f *freeTextAnswer) setQuestionID(questionID string) {
	f.QuestionID = questionID
}

func (f *freeTextAnswer) questionID() string {
	return f.QuestionID
}

func (f *freeTextAnswer) unmarshalMapFromClient(data dataMap) error {

	// a free text answer can be represented as an array of answers
	// containing a single answer object, or the answer object itself.
	answers, err := data.getInterfaceSlice("answers")
	if err != nil {
		return err
	} else if len(answers) == 0 {
		answers = []interface{}{data}
	} else if len(answers) != 1 {
		return fmt.Errorf("Expected exactly 1 entry for free text answer but got %d", len(answers))
	}

	answerMap, err := getDataMap(answers[0])
	if err != nil {
		return err
	}

	if answerMap.requiredKeys("free_text_answer", "answer_text"); err != nil {
		return err
	}

	f.QuestionID = answerMap.mustGetString("question_id")
	f.Text = answerMap.mustGetString("answer_text")

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

func (f *freeTextAnswer) marshalEmptyJSONForClient() ([]byte, error) {
	return emptyTextAnswer(sanitizeQuestionID(f.QuestionID))
}

func (f *freeTextAnswer) marshalJSONForClient() ([]byte, error) {
	clientJSON := &textAnswerClientJSON{
		QuestionID: sanitizeQuestionID(f.QuestionID),
		Items: []*textAnswerClientJSONItem{
			{
				Text: f.Text,
			},
		},
	}

	return json.Marshal(clientJSON)
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

package manager

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

func TestFreeText_JSONMarshaling(t *testing.T) {
	expected := `{"question_id":"43309","potential_answers":[{"answer_text":"No not really..."}]}`

	f := &freeTextAnswer{
		QuestionID: "43309",
		Text:       "No not really...",
	}

	data, err := f.marshalJSONForClient()
	if err != nil {
		t.Fatal(err)
	}

	if res := bytes.Compare(data, []byte(expected)); res != 0 {
		t.Fatalf("Expected `%s`, but got `%s`", expected, string(data))
	}
}

func TestFreeText_ProtobufTransform(t *testing.T) {
	f := &freeTextAnswer{
		QuestionID: "12345",
		Text:       "hello",
	}

	pb, err := f.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	ft, ok := pb.(*intake.FreeTextPatientAnswer)
	if !ok {
		t.Fatalf("Expected type intake.FreeTextPatientAnswer but got %T", pb)
	}

	if *ft.Text != "hello" {
		t.Fatal("data in protocol buffer format doesn't match model")
	}
}

func TestFreeText_ProtobufUnmarshal(t *testing.T) {
	ft := &intake.FreeTextPatientAnswer{
		Text: proto.String("hello"),
	}

	data, err := proto.Marshal(ft)
	if err != nil {
		t.Fatal(err)
	}

	var f freeTextAnswer
	if err := f.unmarshalProtobuf(data); err != nil {
		t.Fatal(err)
	} else if f.Text != "hello" {
		t.Fatal("Data doesn't match when unmarshalling from protobuf")
	}
}

func TestFreeText_DataMapUnmarshal(t *testing.T) {
	clientJSON := `
	{
		"answers": [{
			"answer_id": "64406",
			"question_id": "43309",
			"potential_answer_id": null,
			"answer_text": "Testing free text."
		}]
	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(clientJSON), &data); err != nil {
		t.Fatal(err)
	}

	var f freeTextAnswer
	if err := f.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	} else if f.Text != "Testing free text." {
		t.Fatal("Data unmarshalled from client doesn't match client representation")
	}

	// alternate representation
	clientJSON = `
	{
		"answer_id": "64406",
		"question_id": "43309",
		"potential_answer_id": null,
		"answer_text": "Testing free text."
	}`

	if err := json.Unmarshal([]byte(clientJSON), &data); err != nil {
		t.Fatal(err)
	}

	f = freeTextAnswer{}
	if err := f.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	} else if f.Text != "Testing free text." {
		t.Fatal("Data unmarshalled from client doesn't match client representation")
	}
}

func TestFreeText_equals(t *testing.T) {
	f := &freeTextAnswer{
		QuestionID: "12345",
		Text:       "hello",
	}

	if !f.equals(f) {
		t.Fatal("Answer expected to match self")
	}

	// answer with different text shouldn't match
	other := &freeTextAnswer{
		QuestionID: "12345",
		Text:       "dg",
	}

	if f.equals(other) {
		t.Fatal("Answer not expected to match")
	}
}
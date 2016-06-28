package manager

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
	"github.com/sprucehealth/backend/libs/test"
)

func TestAnswerData_unmarshalProtobuf(t *testing.T) {
	acAnswer := &intake.AutocompletePatientAnswer{
		Answers: []string{
			"Hello",
			"Hello1",
		},
	}

	data, err := proto.Marshal(acAnswer)
	if err != nil {
		t.Fatal(err)
	}

	ad := &intake.PatientAnswerData{
		Type: intake.PatientAnswerData_AUTOCOMPLETE.Enum(),
		Data: data,
	}

	marshalledData, err := proto.Marshal(ad)
	if err != nil {
		t.Fatal(err)
	}

	var a answerData
	if err := a.unmarshalProtobuf(marshalledData); err != nil {
		t.Fatal(err)
	}

	aa, ok := a.answer.(*autocompleteAnswer)
	if !ok {
		t.Fatalf("Expected answer type to be autocomplete but it was %T", a.answer)
	}

	test.Equals(t, 2, len(aa.Answers))
}

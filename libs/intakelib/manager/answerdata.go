package manager

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

// answerData is a container for a patient answer
// and is mainly used to unmarshal an incoming patient
// answer from a client into an object of the type
// as indicated by the type field.
type answerData struct {
	answer patientAnswer
}

func (a *answerData) unmarshalProtobuf(data []byte) error {
	var pd intake.PatientAnswerData
	if err := proto.Unmarshal(data, &pd); err != nil {
		return err
	}

	switch *pd.Type {
	case intake.PatientAnswerData_MULTIPLE_CHOICE:
		a.answer = &multipleChoiceAnswer{}
	case intake.PatientAnswerData_FREE_TEXT:
		a.answer = &freeTextAnswer{}
	case intake.PatientAnswerData_AUTOCOMPLETE:
		a.answer = &autocompleteAnswer{}
	case intake.PatientAnswerData_PHOTO_SECTION:
		a.answer = &photoSectionAnswer{}
	default:
		return fmt.Errorf("Unable to determine answer for type %s", pd.Type)
	}

	return a.answer.unmarshalProtobuf(pd.Data)
}

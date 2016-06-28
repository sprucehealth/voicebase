package manager

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type freeTextQuestion struct {
	*questionInfo
	PlaceholderText string `json:"placeholder_text"`

	answer *freeTextAnswer
}

func (f *freeTextQuestion) staticInfoCopy(context map[string]string) interface{} {
	return &freeTextQuestion{
		questionInfo:    f.questionInfo.staticInfoCopy(context).(*questionInfo),
		PlaceholderText: f.PlaceholderText,
	}
}

func (f *freeTextQuestion) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	var err error
	f.questionInfo, err = populateQuestionInfo(data, parent, questionTypeFreeText.String())
	if err != nil {
		return err
	}

	if data.exists("answers") {
		f.answer = &freeTextAnswer{}
		if err := f.answer.unmarshalMapFromClient(data); err != nil {
			return err
		}
	}

	clientData, err := data.dataMapForKey("additional_fields")
	if err != nil {
		return err
	} else if clientData != nil {
		if clientData.exists("placeholder_text") {
			f.PlaceholderText = clientData.mustGetString("placeholder_text")
		}
	}

	return nil
}

func (f *freeTextQuestion) TypeName() string {
	return questionTypeFreeText.String()
}

// TODO
func (f *freeTextQuestion) validateAnswer(pa patientAnswer) error {
	return nil
}

func (f *freeTextQuestion) setPatientAnswer(answer patientAnswer) error {
	ftAnswer, ok := answer.(*freeTextAnswer)
	if !ok {
		return fmt.Errorf("Expected free text answer instead got %T", answer)
	}

	f.answer = ftAnswer
	return nil
}

func (f *freeTextQuestion) patientAnswer() (patientAnswer, error) {
	if f.answer == nil {
		return nil, errNoAnswerExists
	}
	return f.answer, nil
}

func (f *freeTextQuestion) canPersistAnswer() bool {
	return (f.answer != nil)
}

func (f *freeTextQuestion) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	return f.checkQuestionRequirements(f, f.answer)
}

func (f *freeTextQuestion) marshalAnswerForClient() ([]byte, error) {
	if f.answer == nil {
		return nil, errNoAnswerExists
	}

	if f.visibility() == hidden {
		return f.answer.marshalEmptyJSONForClient()
	}

	return f.answer.marshalJSONForClient()
}

func (f *freeTextQuestion) transformToProtobuf() (proto.Message, error) {
	qInfo, err := transformQuestionInfoToProtobuf(f.questionInfo)
	if err != nil {
		return nil, err
	}

	var freeTextPatientAnswer *intake.FreeTextPatientAnswer
	if f.answer != nil {
		pb, err := f.answer.transformToProtobuf()
		if err != nil {
			return nil, err
		}
		freeTextPatientAnswer = pb.(*intake.FreeTextPatientAnswer)
	}

	return &intake.FreeTextQuestion{
		QuestionInfo:  qInfo.(*intake.CommonQuestionInfo),
		Placeholder:   proto.String(f.PlaceholderText),
		PatientAnswer: freeTextPatientAnswer,
	}, nil
}

func (q *freeTextQuestion) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	b.WriteString(indentAtDepth(indent, depth) + q.layoutUnitID() + ": " + q.Type + " | " + q.v.String() + "\n")
	b.WriteString(indentAtDepth(indent, depth) + "Q: " + q.Title)
	if q.Subtitle != "" {
		b.WriteString("\n")
		b.WriteString(indentAtDepth(indent, depth) + q.Subtitle)
	}
	if q.answer != nil {
		b.WriteString(q.answer.stringIndent(indent, depth))
	}

	return b.String()
}

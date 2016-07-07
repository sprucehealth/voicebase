package manager

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type singleEntryQuestion struct {
	*questionInfo
	PlaceholderText string `json:"placeholder_text"`

	answer *singleEntryAnswer
}

func (f *singleEntryQuestion) staticInfoCopy(context map[string]string) interface{} {
	return &singleEntryQuestion{
		questionInfo:    f.questionInfo.staticInfoCopy(context).(*questionInfo),
		PlaceholderText: f.PlaceholderText,
	}
}

func (f *singleEntryQuestion) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	var err error
	f.questionInfo, err = populateQuestionInfo(data, parent, questionTypeSingleEntry.String())
	if err != nil {
		return err
	}

	answer := dataSource.answerForQuestion(f.id())
	if answer != nil {
		fa, ok := answer.(*singleEntryAnswer)
		if !ok {
			return fmt.Errorf("expected singleEntryAnswer but got %T", answer)
		}
		f.answer = fa
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

func (f *singleEntryQuestion) TypeName() string {
	return questionTypeSingleEntry.String()
}

// TODO
func (f *singleEntryQuestion) validateAnswer(pa patientAnswer) error {
	return nil
}

func (f *singleEntryQuestion) setPatientAnswer(answer patientAnswer) error {
	ftAnswer, ok := answer.(*singleEntryAnswer)
	if !ok {
		return fmt.Errorf("Expected singleEntryAnswer instead got %T", answer)
	}

	f.answer = ftAnswer
	return nil
}

func (f *singleEntryQuestion) patientAnswer() (patientAnswer, error) {
	if f.answer == nil {
		return nil, errNoAnswerExists
	}
	return f.answer, nil
}

func (f *singleEntryQuestion) canPersistAnswer() bool {
	return (f.answer != nil)
}

func (f *singleEntryQuestion) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	return f.checkQuestionRequirements(f, f.answer)
}

func (f *singleEntryQuestion) answerForClient() (interface{}, error) {
	if f.answer == nil {
		return nil, errNoAnswerExists
	}

	return f.answer.transformForClient()
}

func (f *singleEntryQuestion) transformToProtobuf() (proto.Message, error) {
	qInfo, err := transformQuestionInfoToProtobuf(f.questionInfo)
	if err != nil {
		return nil, err
	}

	var singleEntryPatientAnswer *intake.SingleEntryPatientAnswer
	if f.answer != nil {
		pb, err := f.answer.transformToProtobuf()
		if err != nil {
			return nil, err
		}
		singleEntryPatientAnswer = pb.(*intake.SingleEntryPatientAnswer)
	}

	return &intake.SingleEntryQuestion{
		QuestionInfo:  qInfo.(*intake.CommonQuestionInfo),
		Placeholder:   proto.String(f.PlaceholderText),
		PatientAnswer: singleEntryPatientAnswer,
	}, nil
}

func (q *singleEntryQuestion) stringIndent(indent string, depth int) string {
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

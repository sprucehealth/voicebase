package manager

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type autocompleteQuestion struct {
	*questionInfo
	AddText             string `json:"add_text"`
	RemoveButtonText    string `json:"remove_button_text"`
	SaveButtonText      string `json:"save_button_text"`
	AddButtonText       string `json:"add_button_text"`
	PlaceholderText     string `json:"placeholder_text"`
	subquestionsManager *subquestionsManager

	answer *autocompleteAnswer
}

func (a *autocompleteQuestion) staticInfoCopy(context map[string]string) interface{} {
	aCopy := &autocompleteQuestion{
		questionInfo:     a.questionInfo.staticInfoCopy(nil).(*questionInfo),
		AddText:          a.AddText,
		RemoveButtonText: a.RemoveButtonText,
		SaveButtonText:   a.SaveButtonText,
		AddButtonText:    a.AddButtonText,
		PlaceholderText:  a.PlaceholderText,
	}

	if a.subquestionsManager != nil {
		aCopy.subquestionsManager = a.subquestionsManager.staticInfoCopy(context).(*subquestionsManager)
	}

	return aCopy
}

func (a *autocompleteQuestion) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	var err error
	a.questionInfo, err = populateQuestionInfo(data, parent, questionTypeAutocomplete.String())
	if err != nil {
		return err
	}

	clientData, err := data.dataMapForKey("additional_fields")
	if err != nil {
		return err
	} else if clientData != nil {
		a.AddText = clientData.mustGetString("add_text")
		a.RemoveButtonText = clientData.mustGetString("remove_button_text")
		a.SaveButtonText = clientData.mustGetString("save_button_text")
		a.AddButtonText = clientData.mustGetString("add_button_text")
		a.PlaceholderText = clientData.mustGetString("placeholder_text")
	}

	answer := dataSource.answerForQuestion(a.id())
	if answer != nil {
		aa, ok := answer.(*autocompleteAnswer)
		if !ok {
			return fmt.Errorf("expected autocompleteAnswer but got %T", answer)
		}
		a.answer = aa
	}

	if data.exists("subquestions_config") {
		a.subquestionsManager = newSubquestionsManagerForQuestion(a, dataSource)
		subquestionsConfig, err := data.dataMapForKey("subquestions_config")
		if err != nil {
			return err
		}

		if err := a.subquestionsManager.unmarshalMapFromClient(subquestionsConfig); err != nil {
			return err
		}
	}

	return nil
}

func (a *autocompleteQuestion) TypeName() string {
	return questionTypeAutocomplete.String()
}

// TODO
func (a *autocompleteQuestion) validateAnswer(pa patientAnswer) error {
	return nil
}

func (a *autocompleteQuestion) setPatientAnswer(answer patientAnswer) error {
	acAnswer, ok := answer.(*autocompleteAnswer)
	if !ok {
		return fmt.Errorf("Expected an autocomplete answer instead got %T", answer)
	}

	// ensure that none of the answers entered are an empty string
	for _, aItem := range acAnswer.Answers {
		if aItem.text() == "" {
			return errors.New("Cannot have any answer item with an empty string")
		}

		if a.subquestionsManager != nil {
			// transfer ownership of the subscreens if the answers still match
			subscreens := a.subquestionsManager.subscreensForAnswer(aItem)
			aItem.setSubscreens(subscreens)
		}
	}

	a.answer = acAnswer

	if a.subquestionsManager != nil {
		a.subquestionsManager.inflateSubscreensForPatientAnswer()
	}

	return nil
}

func (a *autocompleteQuestion) patientAnswer() (patientAnswer, error) {
	if a.answer == nil {
		return nil, errNoAnswerExists
	}
	return a.answer, nil
}

func (a *autocompleteQuestion) canPersistAnswer() bool {
	return (a.answer != nil)
}

func (a *autocompleteQuestion) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	if a.visibility() == hidden {
		return true, nil
	}

	answerExists := a.answer != nil && !a.answer.isEmpty()

	if !a.Required {
		return true, nil
	} else if !answerExists {
		return false, errQuestionRequirement
	}

	if answerExists {
		// check to ensure that the requirements of each of the subscreens
		// for each answer selection are also met
		for _, aItem := range a.answer.Answers {
			for _, sItem := range aItem.subscreens() {
				if res, err := sItem.requirementsMet(dataSource); err != nil || !res {
					return false, errSubQuestionRequirements
				}
			}
		}
	}

	return true, nil
}

func (a *autocompleteQuestion) answerForClient() (interface{}, error) {
	if a.answer == nil {
		return nil, errNoAnswerExists
	}

	return a.answer.transformForClient()
}

func (a *autocompleteQuestion) transformToProtobuf() (proto.Message, error) {
	qInfo, err := transformQuestionInfoToProtobuf(a.questionInfo)
	if err != nil {
		return nil, err
	}

	var autocompletePatientAnswer *intake.AutocompletePatientAnswer
	if a.answer != nil {
		pb, err := a.answer.transformToProtobuf()
		if err != nil {
			return nil, err
		}
		autocompletePatientAnswer = pb.(*intake.AutocompletePatientAnswer)
	}

	return &intake.AutocompleteQuestion{
		QuestionInfo:     qInfo.(*intake.CommonQuestionInfo),
		PlaceholderText:  proto.String(a.PlaceholderText),
		SaveButtonText:   proto.String(a.SaveButtonText),
		AddButtonText:    proto.String(a.AddButtonText),
		AddText:          proto.String(a.AddText),
		RemoveButtonText: proto.String(a.RemoveButtonText),
		PatientAnswer:    autocompletePatientAnswer,
	}, nil
}

func (q *autocompleteQuestion) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	b.WriteString(indent + q.layoutUnitID() + ": " + q.Type + " | " + q.v.String() + "\n")
	b.WriteString(indent + "Q: " + q.Title)
	if q.Subtitle != "" {
		b.WriteString("\n")
		b.WriteString(indentAtDepth(indent, depth) + q.Subtitle)
	}
	if q.answer != nil {
		b.WriteString(indent + q.answer.stringIndent(indent, depth))
	}

	return b.String()
}

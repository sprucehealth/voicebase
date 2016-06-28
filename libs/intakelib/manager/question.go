package manager

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type questionType string

const (
	questionTypeMultipleChoice   questionType = "q_type_multiple_choice"
	questionTypeSingleSelect     questionType = "q_type_single_select"
	questionTypeSegmentedControl questionType = "q_type_segmented_control"
	questionTypeFreeText         questionType = "q_type_free_text"
	questionTypeAutocomplete     questionType = "q_type_autocomplete"
	questionTypePhoto            questionType = "q_type_photo_section"
	questionTypeSingleEntry      questionType = "q_type_single_entry"
)

var (
	errNoAnswerExists          = errors.New("Answer doesn't exist for question")
	errSubQuestionRequirements = errors.New("Subquestion requirements not met")
	errQuestionRequirement     = errors.New("Please answer the question to continue.")
)

func (q questionType) String() string {
	return string(q)
}

// questionTypeToProtoBufType is a mapping of the supported questionTypes to their corresponding
// protobuf types to be used when transforming the question into serialized form to send to the client.
var questionTypeToProtoBufType = map[string]*intake.QuestionData_Type{
	questionTypeMultipleChoice.String():   intake.QuestionData_MULTIPLE_CHOICE.Enum(),
	questionTypeSingleSelect.String():     intake.QuestionData_MULTIPLE_CHOICE.Enum(),
	questionTypeSegmentedControl.String(): intake.QuestionData_MULTIPLE_CHOICE.Enum(),
	questionTypeFreeText.String():         intake.QuestionData_FREE_TEXT.Enum(),
	questionTypeSingleEntry.String():      intake.QuestionData_FREE_TEXT.Enum(),
	questionTypeAutocomplete.String():     intake.QuestionData_AUTOCOMPLETE.Enum(),
	questionTypePhoto.String():            intake.QuestionData_PHOTO_SECTION.Enum(),
}

func init() {
	mustRegisterQuestion(questionTypeMultipleChoice.String(), &multipleChoiceQuestion{})
	mustRegisterQuestion(questionTypeSingleSelect.String(), &multipleChoiceQuestion{})
	mustRegisterQuestion(questionTypeSegmentedControl.String(), &multipleChoiceQuestion{})
	mustRegisterQuestion(questionTypeFreeText.String(), &freeTextQuestion{})
	mustRegisterQuestion(questionTypeSingleEntry.String(), &freeTextQuestion{})
	mustRegisterQuestion(questionTypeAutocomplete.String(), &autocompleteQuestion{})
	mustRegisterQuestion(questionTypePhoto.String(), &photoQuestion{})
}

// questionInfo represents the common properties included in objects of
// all question types.
type questionInfo struct {
	ID             string     `json:"question_id"`
	LayoutUnitID   string     `json:"-"`
	Type           string     `json:"type"`
	Title          string     `json:"question_title"`
	TitleHasTokens bool       `json:"question_title_has_tokens"`
	Subtitle       string     `json:"question_subtext"`
	Required       bool       `json:"required"`
	Cond           condition  `json:"condition"`
	Parent         layoutUnit `json:"-"`
	Popup          *infoPopup `json:"popup"`
	Prefilled      bool       `json:"prefilled_with_previous_answers"`

	v       visibility
	parentQ question
}

func (q *questionInfo) staticInfoCopy(context map[string]string) interface{} {
	qCopy := &questionInfo{
		ID:             q.ID,
		LayoutUnitID:   q.LayoutUnitID,
		Type:           q.Type,
		Subtitle:       q.Subtitle,
		Required:       q.Required,
		Parent:         q.Parent,
		TitleHasTokens: q.TitleHasTokens,
		Prefilled:      q.prefilled(),
	}

	if q.TitleHasTokens {
		qCopy.Title = processTokenInString(q.Title, context["answer"])
	} else {
		qCopy.Title = q.Title
	}

	if q.Popup != nil {
		qCopy.Popup = q.Popup.staticInfoCopy(context).(*infoPopup)
	}

	if q.Cond != nil {
		qCopy.Cond = q.Cond.staticInfoCopy(context).(condition)
	}
	return qCopy
}

func (q *questionInfo) id() string {
	return q.ID
}

func (q *questionInfo) setID(id string) {
	q.ID = id
}

func (q *questionInfo) prefilled() bool {
	return q.Prefilled
}

func (q *questionInfo) layoutParent() layoutUnit {
	return q.Parent
}

func (q *questionInfo) setLayoutParent(node layoutUnit) {
	q.Parent = node
}

func (q *questionInfo) condition() condition {
	return q.Cond
}

func (q *questionInfo) setCondition(cond condition) {
	q.Cond = cond
}

func (q *questionInfo) children() []layoutUnit {
	return nil
}

func (q *questionInfo) descriptor() string {
	return questionDescriptor
}

func (q *questionInfo) setLayoutUnitID(str string) {
	q.LayoutUnitID = str
}

func (q *questionInfo) layoutUnitID() string {
	return q.LayoutUnitID
}

func (q *questionInfo) setVisibility(v visibility) {
	q.v = v
}

func (q *questionInfo) visibility() visibility {
	return q.v
}

func (q *questionInfo) setParentQuestion(parentQ question) {
	q.parentQ = parentQ
}

func (q *questionInfo) parentQuestion() question {
	return q.parentQ
}

func (q *questionInfo) String() string {
	return fmt.Sprintf("  %s: %s | %s\n    Q: %s", q.layoutUnitID(), q.Type, q.v, q.Title)
}

func (q *questionInfo) stringIndent(indent string, depth int) string {
	return fmt.Sprintf("%s%s: %s | %s\n%sQ: %s", indentAtDepth(indent, depth), q.layoutUnitID(), q.Type, q.v, indentAtDepth(indent, depth+1), q.Title)
}

func transformQuestionInfoToProtobuf(q *questionInfo) (proto.Message, error) {
	var pInfo *intake.InfoPopup
	if q.Popup != nil {
		pData, err := q.Popup.transformToProtobuf()
		if err != nil {
			return nil, err
		}
		pInfo = pData.(*intake.InfoPopup)
	}

	qInfoProtoBuf := &intake.CommonQuestionInfo{
		Title:      proto.String(q.Title),
		Subtitle:   proto.String(q.Subtitle),
		Id:         proto.String(q.ID),
		InfoPopup:  pInfo,
		IsRequired: proto.Bool(q.Required),
	}

	return qInfoProtoBuf, nil
}

func populateQuestionInfo(data dataMap, parent layoutUnit, typeName string) (*questionInfo, error) {
	if err := data.requiredKeys(typeName,
		"question_id", "type", "question_title"); err != nil {
		return nil, err
	}

	q := &questionInfo{
		ID:             data.mustGetString("question_id"),
		Type:           data.mustGetString("type"),
		Title:          data.mustGetString("question_title"),
		TitleHasTokens: data.mustGetBool("question_title_has_tokens"),
		Subtitle:       data.mustGetString("question_subtext"),
		Required:       data.mustGetBool("required"),
		Prefilled:      data.mustGetBool("prefilled_with_previous_answers"),
		Parent:         parent,
	}

	additionalFieldsMap, err := data.dataMapForKey("additional_fields")
	if err != nil {
		return nil, err
	} else if additionalFieldsMap != nil {
		q.Popup, err = populatePopup(additionalFieldsMap)
		if err != nil {
			return nil, err
		}
	}

	conditionDataMap, err := data.dataMapForKey("condition")
	if err != nil {
		return nil, err
	} else if conditionDataMap != nil {
		q.Cond, err = getCondition(conditionDataMap)
		if err != nil {
			return nil, err
		}
	}

	return q, nil
}

// checkQuestionRequirements returns an error if the requirements are not met, and nil otherwise.
func (qi *questionInfo) checkQuestionRequirements(q question, pa patientAnswer) (bool, error) {
	// requirements are met if the question is hidden.
	if q.visibility() == hidden {
		return true, nil
	}

	// requirements are met if the question is optional and doesn't have an answer set.
	if !qi.Required {
		return true, nil
	}

	// a required question is expected to have an answer set
	pa, err := q.patientAnswer()
	if err != nil {
		return false, errQuestionRequirement
	} else if pa.isEmpty() {
		return false, errQuestionRequirement
	}

	return true, nil
}

package manager

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

const (
	answerTypeOption    = "a_type_multiple_choice"
	answerTypeNone      = "a_type_multiple_choice_none"
	answerTypeOther     = "a_type_multiple_choice_other_free_text"
	answerTypeSegmented = "a_type_segmented_control"
)

// byPotentialAnswerOrder determines the ordering of top level answers
// based on the ordering of the potential answers within the question structure.
// This is used to provide a consistent ordering of answers to the user,
// particulary when the ordering of answers determines the ordering of the subscreens
// shown to the user.
type byPotentialAnswerOrder struct {
	answers            []topLevelAnswerItem
	potentialAnswerMap map[string]*potentialAnswer
}

func (a *byPotentialAnswerOrder) Len() int { return len(a.answers) }
func (a *byPotentialAnswerOrder) Swap(i, j int) {
	a.answers[i], a.answers[j] = a.answers[j], a.answers[i]
}
func (a *byPotentialAnswerOrder) Less(i, j int) bool {
	return a.potentialAnswerMap[a.answers[i].potentialAnswerID()].position < a.potentialAnswerMap[a.answers[j].potentialAnswerID()].position
}

type potentialAnswer struct {
	ID              string     `json:"potential_answer_id"`
	Text            string     `json:"potential_answer"`
	Summary         string     `json:"potential_answer_summary"`
	Type            string     `json:"answer_type"`
	Popup           *infoPopup `json:"popup"`
	PlaceholderText string     `json:"placeholder_text"`

	position int
}

func (p *potentialAnswer) staticInfoCopy(context map[string]string) interface{} {
	return &potentialAnswer{
		ID:              p.ID,
		Text:            p.Text,
		Summary:         p.Summary,
		Type:            p.Type,
		Popup:           p.Popup,
		PlaceholderText: p.PlaceholderText,
		position:        p.position,
	}
}

func (p *potentialAnswer) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("potential_answer",
		"potential_answer_id", "potential_answer", "answer_type"); err != nil {
		return err
	}

	p.ID = data.mustGetString("potential_answer_id")
	p.Text = data.mustGetString("potential_answer")
	p.Summary = data.mustGetString("potential_answer_summary")
	p.Type = data.mustGetString("answer_type")

	clientData, err := data.dataMapForKey("client_data")
	if err != nil {
		return err
	} else if clientData != nil {
		p.Popup, err = populatePopup(clientData)
		if err != nil {
			return err
		}

		p.PlaceholderText = clientData.mustGetString("placeholder_text")
	}

	if p.Type == answerTypeOther && p.PlaceholderText == "" {
		p.PlaceholderText = "Type to add another"
	}

	return nil
}

type titleCount struct {
	Title string `json:"title"`
	Count int    `json:"count"`
}

func (t *titleCount) staticInfoCopy(context map[string]string) interface{} {
	return &titleCount{
		Title: t.Title,
		Count: t.Count,
	}
}

func (t *titleCount) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("title_count", "title", "count"); err != nil {
		return err
	}

	t.Title = data.mustGetString("title")
	t.Count = data.mustGetInt("count")

	return nil
}

type multipleChoiceQuestion struct {
	*questionInfo
	PotentialAnswers    []*potentialAnswer `json:"potential_answers"`
	AnswerGroups        []*titleCount      `json:"answer_groups"`
	subquestionsManager *subquestionsManager

	potentialAnswerMap map[string]*potentialAnswer
	answer             *multipleChoiceAnswer
}

func (m *multipleChoiceQuestion) staticInfoCopy(context map[string]string) interface{} {
	qCopy := &multipleChoiceQuestion{
		questionInfo:       m.questionInfo.staticInfoCopy(context).(*questionInfo),
		PotentialAnswers:   make([]*potentialAnswer, len(m.PotentialAnswers)),
		AnswerGroups:       make([]*titleCount, len(m.AnswerGroups)),
		potentialAnswerMap: make(map[string]*potentialAnswer),
	}

	if m.subquestionsManager != nil {
		qCopy.subquestionsManager = m.subquestionsManager.staticInfoCopy(context).(*subquestionsManager)
	}

	for i, pa := range m.PotentialAnswers {
		qCopy.PotentialAnswers[i] = pa.staticInfoCopy(context).(*potentialAnswer)
	}

	for i, ag := range m.AnswerGroups {
		qCopy.AnswerGroups[i] = ag.staticInfoCopy(context).(*titleCount)
	}

	for k, v := range m.potentialAnswerMap {
		qCopy.potentialAnswerMap[k] = v
	}

	return qCopy
}

func (m *multipleChoiceQuestion) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys(
		questionTypeMultipleChoice.String(), "potential_answers"); err != nil {
		return err
	}

	var err error
	m.questionInfo, err = populateQuestionInfo(data, parent, questionTypeMultipleChoice.String())
	if err != nil {
		return err
	}

	potentialAnswers, err := data.getInterfaceSlice("potential_answers")
	if err != nil {
		return err
	}

	m.PotentialAnswers = make([]*potentialAnswer, len(potentialAnswers))
	m.potentialAnswerMap = make(map[string]*potentialAnswer, len(potentialAnswers))
	for i, potentialAnswerVal := range potentialAnswers {
		potentialAnswerMap, err := getDataMap(potentialAnswerVal)
		if err != nil {
			return err
		}

		pa := &potentialAnswer{
			position: i,
		}
		if err := pa.unmarshalMapFromClient(potentialAnswerMap); err != nil {
			return err
		}
		m.PotentialAnswers[i] = pa
		m.potentialAnswerMap[pa.ID] = pa
	}

	if data.exists("answers") {
		m.answer = &multipleChoiceAnswer{}
		if err := m.answer.unmarshalMapFromClient(data); err != nil {
			return err
		}
	}

	clientData, err := data.dataMapForKey("additional_fields")
	if err != nil {
		return err
	} else if clientData != nil {

		if clientData.exists("answer_groups") {

			answerGroups, err := clientData.getInterfaceSlice("answer_groups")
			if err != nil {
				return err
			}

			m.AnswerGroups = make([]*titleCount, len(answerGroups))
			for i, titleCountVal := range answerGroups {
				titleCountMap, err := getDataMap(titleCountVal)
				if err != nil {
					return err
				}

				tc := &titleCount{}
				if err := tc.unmarshalMapFromClient(titleCountMap); err != nil {
					return err
				}
				m.AnswerGroups[i] = tc
			}
		}
	}

	subquestionsConfig, err := data.dataMapForKey("subquestions_config")
	if err != nil {
		return err
	} else if subquestionsConfig != nil {
		m.subquestionsManager = newSubquestionsManagerForQuestion(m, dataSource)
		if err := m.subquestionsManager.unmarshalMapFromClient(subquestionsConfig); err != nil {
			return err
		}
	}

	return nil
}

func (q *multipleChoiceQuestion) TypeName() string {
	return questionTypeMultipleChoice.String()
}

// TODO
func (q *multipleChoiceQuestion) validateAnswer(pa patientAnswer) error {
	return nil
}

func (q *multipleChoiceQuestion) setPatientAnswer(answer patientAnswer) error {

	mcqAnswer, ok := answer.(*multipleChoiceAnswer)
	if !ok {
		return fmt.Errorf("Expected multiple choice answer instead got %T for question %s", answer, q.LayoutUnitID)
	}

	switch q.Type {
	case questionTypeSingleSelect.String(), questionTypeSegmentedControl.String():
		if len(mcqAnswer.Answers) > 1 {
			return fmt.Errorf("Cannot have more than one answer selection for single select question %s", q.LayoutUnitID)
		}
	}

	// first validate to ensure that each answer selection actually exists in the
	// potential answer set
	for _, aItem := range mcqAnswer.Answers {
		if _, ok := q.potentialAnswerMap[aItem.potentialAnswerID()]; !ok {
			return fmt.Errorf("potential_answer_id %s is not a valid selection for question %s", aItem.potentialAnswerID(), q.LayoutUnitID)
		}
	}

	// order the answer selection from the client based on the ordering of the
	// potential answers to ensure consistent order of subscreens (if there are subquestions)
	sort.Sort(&byPotentialAnswerOrder{
		answers:            mcqAnswer.Answers,
		potentialAnswerMap: q.potentialAnswerMap,
	})

	for _, aItem := range mcqAnswer.Answers {
		selection := aItem.(*multipleChoiceAnswerSelection)
		potentialAnswerItem := q.potentialAnswerMap[aItem.potentialAnswerID()]
		selection.PotentialAnswerText = potentialAnswerItem.Text

		switch potentialAnswerItem.Type {
		case answerTypeOption:
			if selection.Text != "" {
				return fmt.Errorf("cannot select regular option and specify custom text for question %s", q.LayoutUnitID)
			}
		case answerTypeNone:
			if len(mcqAnswer.Answers) > 1 {
				return fmt.Errorf("cannot select more than 1 option if none of the above is selected for question %s", q.LayoutUnitID)
			}
		case answerTypeOther:
			if potentialAnswerItem.Text == "" {
				return fmt.Errorf("cannot select other free text and have no text entry corresponding to it for question %s", q.LayoutUnitID)
			}
		}

		if q.subquestionsManager != nil {
			// transfer ownership of the subscreens if the answers still match
			subscreens := q.subquestionsManager.subscreensForAnswer(aItem)
			aItem.setSubscreens(subscreens)
		}

	}

	q.answer = mcqAnswer

	if q.subquestionsManager != nil {
		if err := q.subquestionsManager.inflateSubscreensForPatientAnswer(); err != nil {
			return err
		}
	}
	return nil
}

func (a *multipleChoiceQuestion) patientAnswer() (patientAnswer, error) {
	if a.answer == nil {
		return nil, errNoAnswerExists
	}
	return a.answer, nil
}

func (a *multipleChoiceQuestion) canPersistAnswer() bool {
	return (a.answer != nil)
}

func (q *multipleChoiceQuestion) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	if q.visibility() == hidden {
		return true, nil
	}

	answerExists := q.answer != nil && !q.answer.isEmpty()

	if !q.Required {
		return true, nil
	} else if !answerExists {
		return false, errQuestionRequirement
	}

	// check to ensure that the requirements of each of the subscreens
	// for each answer selection are also met
	if answerExists {
		for _, selectionItem := range q.answer.Answers {
			for _, sItem := range selectionItem.subscreens() {
				if res, err := sItem.requirementsMet(dataSource); err != nil || !res {
					return false, errSubQuestionRequirements
				}
			}
		}
	}

	return true, nil
}

func (q *multipleChoiceQuestion) marshalAnswerForClient() ([]byte, error) {
	if q.answer == nil {
		return nil, errNoAnswerExists
	}

	if q.visibility() == hidden {
		return q.answer.marshalEmptyJSONForClient()
	}

	return q.answer.marshalJSONForClient()
}

func (q *multipleChoiceQuestion) transformToProtobuf() (proto.Message, error) {
	qInfo, err := transformQuestionInfoToProtobuf(q.questionInfo)
	if err != nil {
		return nil, err
	}

	multipleChoiceProtoBuf := &intake.MultipleChoiceQuestion{
		QuestionInfo: qInfo.(*intake.CommonQuestionInfo),
	}

	var config *intake.MultipleChoiceQuestion_Config
	switch q.Type {
	case questionTypeMultipleChoice.String():
		config = intake.MultipleChoiceQuestion_MULTIPLE_CHOICE.Enum()
	case questionTypeSingleSelect.String():
		config = intake.MultipleChoiceQuestion_SINGLE_SELECT.Enum()
	case questionTypeSegmentedControl.String():
		config = intake.MultipleChoiceQuestion_SEGMENTED_CONTROL.Enum()
	default:
		return nil, fmt.Errorf("Unable to determine config for %s", q.Type)
	}
	multipleChoiceProtoBuf.Config = config

	multipleChoiceProtoBuf.PotentialAnswers = make([]*intake.MultipleChoiceQuestion_PotentialAnswer, len(q.PotentialAnswers))
	for i, pa := range q.PotentialAnswers {
		transformedPA := &intake.MultipleChoiceQuestion_PotentialAnswer{
			Text:            proto.String(pa.Text),
			Id:              proto.String(pa.ID),
			PlaceholderText: proto.String(pa.PlaceholderText),
		}

		if pa.Popup != nil {
			ipInfo, err := pa.Popup.transformToProtobuf()
			if err != nil {
				return nil, err
			}
			transformedPA.InfoPopup = ipInfo.(*intake.InfoPopup)
		}

		switch pa.Type {
		case answerTypeOption, answerTypeSegmented:
			transformedPA.Type = intake.MultipleChoiceQuestion_PotentialAnswer_OPTION.Enum()
		case answerTypeNone:
			transformedPA.Type = intake.MultipleChoiceQuestion_PotentialAnswer_NONE_OF_THE_ABOVE.Enum()
		case answerTypeOther:
			transformedPA.Type = intake.MultipleChoiceQuestion_PotentialAnswer_OTHER_FREE_TEXT.Enum()
		default:
			return nil, fmt.Errorf("Unable to determine potential answer type %s", pa.Type)
		}

		multipleChoiceProtoBuf.PotentialAnswers[i] = transformedPA
	}

	multipleChoiceProtoBuf.Groups = make([]*intake.MultipleChoiceQuestion_TitleCount, len(q.AnswerGroups))
	for i, ag := range q.AnswerGroups {
		multipleChoiceProtoBuf.Groups[i] = &intake.MultipleChoiceQuestion_TitleCount{
			Title: proto.String(ag.Title),
			Count: proto.Int(ag.Count),
		}
	}

	if q.answer != nil {
		pb, err := q.answer.transformToProtobuf()
		if err != nil {
			return nil, err
		}
		multipleChoiceProtoBuf.PatientAnswer = pb.(*intake.MultipleChoicePatientAnswer)
	}

	return multipleChoiceProtoBuf, nil

}

func (q *multipleChoiceQuestion) stringIndent(indent string, depth int) string {
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

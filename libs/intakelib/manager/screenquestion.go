package manager

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type questionScreen struct {
	*screenInfo
	Questions                          []question `json:"questions"`
	ContentHeaderTitle                 string     `json:"header_title"`
	ContentHeaderTitleHasTokens        bool       `json:"header_title_has_tokens"`
	ContentHeaderSubtitle              string     `json:"header_subtitle"`
	Popup                              *infoPopup `json:"popup"`
	RequiresAtleastOneQuestionAnswered bool       `json:"requires_at_least_one_question_answered"`

	subscreensMap map[string][]screen
	allSubscreens []screen
}

func (q *questionScreen) staticInfoCopy(context map[string]string) interface{} {
	qsCopy := &questionScreen{
		screenInfo:                         q.screenInfo.staticInfoCopy(context).(*screenInfo),
		Questions:                          make([]question, len(q.Questions)),
		ContentHeaderTitleHasTokens:        q.ContentHeaderTitleHasTokens,
		ContentHeaderSubtitle:              q.ContentHeaderSubtitle,
		RequiresAtleastOneQuestionAnswered: q.RequiresAtleastOneQuestionAnswered,
	}

	if qsCopy.ContentHeaderTitleHasTokens {
		qsCopy.ContentHeaderTitle = processTokenInString(q.ContentHeaderTitle, context["answer"])
	} else {
		qsCopy.ContentHeaderTitle = q.ContentHeaderTitle
	}

	if q.Popup != nil {
		qsCopy.Popup = q.Popup.staticInfoCopy(context).(*infoPopup)
	}

	for i, qItem := range q.Questions {
		qsCopy.Questions[i] = qItem.staticInfoCopy(context).(question)
		qsCopy.Questions[i].setLayoutParent(qsCopy)
	}

	return qsCopy
}

func (q *questionScreen) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {

	if err := data.requiredKeys(screenTypeQuestion.String(), "questions"); err != nil {
		return err
	}

	var err error
	q.screenInfo, err = populateScreenInfo(data, parent)
	if err != nil {
		return err
	}

	// get the title from the parent if the parent has a title
	p, ok := parent.(titler)
	if ok {
		q.screenInfo.Title = p.title()
	}

	q.ContentHeaderTitle = data.mustGetString("header_title")
	q.ContentHeaderTitleHasTokens = data.mustGetBool("header_title_has_tokens")
	q.ContentHeaderSubtitle = data.mustGetString("header_subtitle")

	clientData, err := data.dataMapForKey("client_data")
	if err != nil {
		return err
	} else if clientData != nil {
		q.Popup, err = populatePopup(clientData)
		if err != nil {
			return err
		}
		q.RequiresAtleastOneQuestionAnswered = clientData.mustGetBool("requires_at_least_one_question_answered")
	}

	questions, err := data.getInterfaceSlice("questions")
	if err != nil {
		return err
	}

	q.Questions = make([]question, len(questions))
	for i, qVal := range questions {
		questionMap, err := getDataMap(qVal)
		if err != nil {
			return err
		}

		q.Questions[i], err = getQuestion(questionMap, q, dataSource)
		if err != nil {
			return err
		}
	}

	q.subscreensMap = make(map[string][]screen)

	return nil
}

func (q *questionScreen) TypeName() string {
	return screenTypeQuestion.String()
}

func (q *questionScreen) children() []layoutUnit {
	children := make([]layoutUnit, len(q.Questions))
	for i, qs := range q.Questions {
		children[i] = qs
	}

	return children
}

func (q *questionScreen) questions() []question {
	return q.Questions
}

func (s *questionScreen) setVisibility(v visibility) {
	s.v = v
}

func (s *questionScreen) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	// requirements are met if the screen is hidden
	if s.visibility() == hidden {
		return true, nil
	}

	var atLeastOneQAnswered bool
	// ensure that the requirements for all questions have been met
	for _, pq := range s.Questions {
		if res, err := pq.requirementsMet(dataSource); err != nil {
			return res, err
		} else if !res {
			return false, nil
		}

		pa, err := pq.patientAnswer()
		if err == nil && !pa.isEmpty() {
			atLeastOneQAnswered = true
		}
	}

	// at least one of the questions must be answered even if
	// all questions are optional if this configuration is turned on.
	if s.RequiresAtleastOneQuestionAnswered && !atLeastOneQAnswered {
		return false, errors.New("Answer at least one question on the screen to continue.")
	}

	return true, nil
}

func (q *questionScreen) transformToProtobuf() (proto.Message, error) {
	sInfo, err := transformScreenInfoToProtobuf(q.screenInfo)
	if err != nil {
		return nil, err
	}

	var pInfo *intake.InfoPopup
	if q.Popup != nil {
		pData, err := q.Popup.transformToProtobuf()
		if err != nil {
			return nil, err
		}
		pInfo = pData.(*intake.InfoPopup)
	}

	qScreenProtobuf := &intake.QuestionScreen{
		ScreenInfo:             sInfo.(*intake.CommonScreenInfo),
		ContentHeaderTitle:     proto.String(q.ContentHeaderTitle),
		ContentHeaderSubtitle:  proto.String(q.ContentHeaderSubtitle),
		Questions:              make([]*intake.QuestionData, 0, len(q.Questions)),
		ContentHeaderInfoPopup: pInfo,
	}

	for _, qs := range q.Questions {
		qType := questionTypeToProtoBufType[qs.TypeName()]
		if qType == nil {
			return nil, fmt.Errorf("Unable to determine question type: %s", qs.TypeName())
		} else if qs.visibility() == hidden {
			// skip hidden questions as they should not be sent to client
			continue
		}

		transformedQ, err := qs.transformToProtobuf()
		if err != nil {
			return nil, err
		}

		data, err := proto.Marshal(transformedQ.(proto.Message))
		if err != nil {
			return nil, err
		}

		qScreenProtobuf.Questions = append(qScreenProtobuf.Questions, &intake.QuestionData{
			Type: qType,
			Data: data,
		})
	}

	return qScreenProtobuf, nil
}

func (q *questionScreen) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("%s%s: %s | %s", indentAtDepth(indent, depth), q.layoutUnitID(), q.TypeName(), q.v))
	if q.ContentHeaderTitle != "" {
		b.WriteString("\n")
		b.WriteString(indentAtDepth(indent, depth) + q.ContentHeaderTitle)
	}
	if q.ContentHeaderSubtitle != "" {
		b.WriteString("\n")
		b.WriteString(indentAtDepth(indent, depth) + q.ContentHeaderSubtitle)
	}

	for _, qItem := range q.Questions {
		b.WriteString("\n")
		b.WriteString(qItem.stringIndent(indent, depth+1))
	}
	return b.String()
}

// subscreens returns a flat and ordered list of the subscreens to show
// to the user. they are ordered in the order of the questions on the screen.
func (s *questionScreen) subscreens() []screen {
	// clear out the slice and reuse
	s.allSubscreens = s.allSubscreens[:0]

	for _, qItem := range s.Questions {
		s.allSubscreens = append(s.allSubscreens, s.subscreensMap[qItem.layoutUnitID()]...)
	}

	return s.allSubscreens
}

// registerSubscreensForQuestion give the questionScreen the responsibility of ensuring
// that the requirements of the subscreens are met and that they are surfaced for screen navigation.
func (s *questionScreen) registerSubscreensForQuestion(q question, subscreens []screen) {
	s.subscreensMap[q.layoutUnitID()] = make([]screen, len(subscreens))
	for i, subscreen := range subscreens {
		s.subscreensMap[q.layoutUnitID()][i] = subscreen
	}
}

// deregisterSubscreensForQuestion removes the subscreens associated with a particular question.
func (s *questionScreen) deregisterSubscreensForQuestion(q question) {
	s.subscreensMap[q.layoutUnitID()] = nil
}

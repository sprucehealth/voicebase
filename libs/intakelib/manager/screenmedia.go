package manager

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type mediaScreen struct {
	*screenInfo

	ContentHeaderTitle          string     `json:"header_title"`
	ContentHeaderSubtitle       string     `json:"header_subtitle"`
	ContentHeaderTitleHasTokens bool       `json:"header_title_has_tokens"`
	Popup                       *infoPopup `json:"popup"`

	MediaQuestions                     []question `json:"questions"`
	RequiresAtleastOneQuestionAnswered bool       `json:"requires_at_least_one_question_answered"`
}

func (p *mediaScreen) staticInfoCopy(context map[string]string) interface{} {
	pCopy := &mediaScreen{
		screenInfo:                  p.screenInfo.staticInfoCopy(context).(*screenInfo),
		ContentHeaderTitle:          p.ContentHeaderTitle,
		ContentHeaderSubtitle:       p.ContentHeaderSubtitle,
		ContentHeaderTitleHasTokens: p.ContentHeaderTitleHasTokens,
		MediaQuestions:              make([]question, len(p.MediaQuestions)),
	}

	if pCopy.ContentHeaderTitleHasTokens {
		pCopy.ContentHeaderTitle = processTokenInString(p.ContentHeaderTitle, context["answer"])
	} else {
		pCopy.ContentHeaderTitle = p.ContentHeaderTitle
	}

	if p.Popup != nil {
		pCopy.Popup = p.Popup.staticInfoCopy(context).(*infoPopup)
	}

	for i, pq := range p.MediaQuestions {
		pCopy.MediaQuestions[i] = pq.staticInfoCopy(context).(question)
	}

	return pCopy
}

func (q *mediaScreen) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys(screenTypeMedia.String(), "questions", "header_title"); err != nil {
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

	var requiresAtleastOneQuestionAnsweredSpecified bool
	clientDataMap, err := data.dataMapForKey("additional_fields")
	if err != nil {
		return err
	} else if clientDataMap != nil {
		q.Popup, err = populatePopup(clientDataMap)
		if err != nil {
			return err
		}

		if clientDataMap.exists("requires_at_least_one_question_answered") {
			q.RequiresAtleastOneQuestionAnswered = clientDataMap.mustGetBool("requires_at_least_one_question_answered")
			requiresAtleastOneQuestionAnsweredSpecified = true
		}
	}

	if !requiresAtleastOneQuestionAnsweredSpecified {
		// default to true if not specified by the server
		q.RequiresAtleastOneQuestionAnswered = true
	}

	questions, err := data.getInterfaceSlice("questions")
	if err != nil {
		return err
	}

	q.MediaQuestions = make([]question, len(questions))
	for i, questionVal := range questions {
		questionMap, err := getDataMap(questionVal)
		if err != nil {
			return err
		}

		q.MediaQuestions[i], err = getQuestion(questionMap, q, dataSource)
		if err != nil {
			return err
		}

		_, ok := q.MediaQuestions[i].(*mediaQuestion)
		if !ok {
			return fmt.Errorf("A photo question screen can only have photo questions. Got %s question type.", q.MediaQuestions[i].TypeName())
		}
	}

	return nil
}

func (q *mediaScreen) TypeName() string {
	return screenTypeMedia.String()
}

func (q *mediaScreen) children() []layoutUnit {
	children := make([]layoutUnit, len(q.MediaQuestions))
	for i, qs := range q.MediaQuestions {
		children[i] = qs
	}

	return children
}

func (q *mediaScreen) questions() []question {
	return q.MediaQuestions
}

// set the questions' hidden state to hidden if the screen is hidden.
func (s *mediaScreen) setVisibility(v visibility) {
	s.v = v
}

func (s *mediaScreen) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	if s.visibility() == hidden {
		return true, nil
	}

	var atLeastOneQAnswered bool
	// ensure that the requirements for all questions have been met
	for _, pq := range s.MediaQuestions {
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

func (q *mediaScreen) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("%s%s: %s | %s", indentAtDepth(indent, depth), q.layoutUnitID(), q.TypeName(), q.v))
	for _, qItem := range q.MediaQuestions {
		b.WriteString("\n")
		b.WriteString(qItem.stringIndent(indent, depth+1))
	}
	return b.String()
}

func (q *mediaScreen) transformToProtobuf() (proto.Message, error) {
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

	mediaQuestions := make([]*intake.MediaSectionQuestion, 0, len(q.MediaQuestions))
	for _, pq := range q.MediaQuestions {

		if pq.visibility() == hidden {
			// skip hidden questions as they should not be sent to the client
			continue
		}
		transformedMediaQuestion, err := pq.transformToProtobuf()
		if err != nil {
			return nil, err
		}

		mediaQuestions = append(mediaQuestions, transformedMediaQuestion.(*intake.MediaSectionQuestion))
	}

	return &intake.MediaScreen{
		ScreenInfo:             sInfo.(*intake.CommonScreenInfo),
		ContentHeaderTitle:     proto.String(q.ContentHeaderTitle),
		ContentHeaderSubtitle:  proto.String(q.ContentHeaderSubtitle),
		MediaQuestions:         mediaQuestions,
		ContentHeaderInfoPopup: pInfo,
		BottomContainer: &intake.MediaScreen_ImageTextBox{
			ImageLink: proto.String("spruce:///icon/lock"),
			Text:      proto.String("Your photos and videos are only accessible to you and your care providers. They are not saved to your phone's camera roll."),
		},
	}, nil
}

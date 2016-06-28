package manager

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type photoScreen struct {
	*screenInfo

	ContentHeaderTitle          string     `json:"header_title"`
	ContentHeaderSubtitle       string     `json:"header_subtitle"`
	ContentHeaderTitleHasTokens bool       `json:"header_title_has_tokens"`
	Popup                       *infoPopup `json:"popup"`

	PhotoQuestions                     []question `json:"questions"`
	RequiresAtleastOneQuestionAnswered bool       `json:"requires_at_least_one_question_answered"`
}

func (p *photoScreen) staticInfoCopy(context map[string]string) interface{} {
	pCopy := &photoScreen{
		screenInfo:                  p.screenInfo.staticInfoCopy(context).(*screenInfo),
		ContentHeaderTitle:          p.ContentHeaderTitle,
		ContentHeaderSubtitle:       p.ContentHeaderSubtitle,
		ContentHeaderTitleHasTokens: p.ContentHeaderTitleHasTokens,
		PhotoQuestions:              make([]question, len(p.PhotoQuestions)),
	}

	if pCopy.ContentHeaderTitleHasTokens {
		pCopy.ContentHeaderTitle = processTokenInString(p.ContentHeaderTitle, context["answer"])
	} else {
		pCopy.ContentHeaderTitle = p.ContentHeaderTitle
	}

	if p.Popup != nil {
		pCopy.Popup = p.Popup.staticInfoCopy(context).(*infoPopup)
	}

	for i, pq := range p.PhotoQuestions {
		pCopy.PhotoQuestions[i] = pq.staticInfoCopy(context).(question)
	}

	return pCopy
}

func (q *photoScreen) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys(screenTypePhoto.String(), "questions", "header_title"); err != nil {
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

	q.PhotoQuestions = make([]question, len(questions))
	for i, questionVal := range questions {
		questionMap, err := getDataMap(questionVal)
		if err != nil {
			return err
		}

		q.PhotoQuestions[i], err = getQuestion(questionMap, q, dataSource)
		if err != nil {
			return err
		}

		_, ok := q.PhotoQuestions[i].(*photoQuestion)
		if !ok {
			return fmt.Errorf("A photo question screen can only have photo questions. Got %s question type.", q.PhotoQuestions[i].TypeName())
		}
	}

	return nil
}

func (q *photoScreen) TypeName() string {
	return screenTypePhoto.String()
}

func (q *photoScreen) children() []layoutUnit {
	children := make([]layoutUnit, len(q.PhotoQuestions))
	for i, qs := range q.PhotoQuestions {
		children[i] = qs
	}

	return children
}

func (q *photoScreen) questions() []question {
	return q.PhotoQuestions
}

// set the questions' hidden state to hidden if the screen is hidden.
func (s *photoScreen) setVisibility(v visibility) {
	s.v = v
}

func (s *photoScreen) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	if s.visibility() == hidden {
		return true, nil
	}

	var atLeastOneQAnswered bool
	// ensure that the requirements for all questions have been met
	for _, pq := range s.PhotoQuestions {
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

func (q *photoScreen) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("%s%s: %s | %s", indentAtDepth(indent, depth), q.layoutUnitID(), q.TypeName(), q.v))
	for _, qItem := range q.PhotoQuestions {
		b.WriteString("\n")
		b.WriteString(qItem.stringIndent(indent, depth+1))
	}
	return b.String()
}

func (q *photoScreen) transformToProtobuf() (proto.Message, error) {
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

	photoQuestions := make([]*intake.PhotoSectionQuestion, 0, len(q.PhotoQuestions))
	for _, pq := range q.PhotoQuestions {

		if pq.visibility() == hidden {
			// skip hidden questions as they should not be sent to the client
			continue
		}
		transformedPhotoQuestion, err := pq.transformToProtobuf()
		if err != nil {
			return nil, err
		}

		photoQuestions = append(photoQuestions, transformedPhotoQuestion.(*intake.PhotoSectionQuestion))
	}

	return &intake.PhotoScreen{
		ScreenInfo:             sInfo.(*intake.CommonScreenInfo),
		ContentHeaderTitle:     proto.String(q.ContentHeaderTitle),
		ContentHeaderSubtitle:  proto.String(q.ContentHeaderSubtitle),
		PhotoQuestions:         photoQuestions,
		ContentHeaderInfoPopup: pInfo,
		BottomContainer: &intake.PhotoScreen_ImageTextBox{
			ImageLink: proto.String("spruce:///icon/lock"),
			Text:      proto.String("Your photos are only accessible to you and your care providers. Photos are not saved to your phone's camera roll."),
		},
	}, nil
}

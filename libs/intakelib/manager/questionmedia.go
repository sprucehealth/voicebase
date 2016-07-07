package manager

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type mediaTip struct {
	Tip        string `json:"tip"`
	TipSubtext string `json:"tip_subtext"`
	TipStyle   string `json:"tip_style"`
}

type mediaSlot struct {
	Name                     string `json:"name"`
	ID                       string `json:"id"`
	Required                 bool   `json:"required"`
	Type                     string `json:"type"`
	OverlayImageLink         string `json:"overlay_image_url"`
	MediaMissingErrorMessage string `json:"media_missing_error_message"`
	InitialCameraDirection   string `json:"initial_camera_direction"`
	FlashState               string `json:"flash"`

	clientPlatform platform
	mediaTip
	Tips map[string]*mediaTip `json:"tips"`
}

func (p *mediaSlot) staticInfoCopy(context map[string]string) interface{} {
	ps := &mediaSlot{
		Name:     p.Name,
		ID:       p.ID,
		Required: p.Required,
		Type:     p.Type,
		mediaTip: mediaTip{
			Tip:        p.Tip,
			TipSubtext: p.TipSubtext,
			TipStyle:   p.TipStyle,
		},
		OverlayImageLink:         p.OverlayImageLink,
		MediaMissingErrorMessage: p.MediaMissingErrorMessage,
		InitialCameraDirection:   p.InitialCameraDirection,
		FlashState:               p.FlashState,
	}

	if p.Tips != nil {
		ps.Tips = make(map[string]*mediaTip, len(p.Tips))
		for name, tip := range p.Tips {
			ps.Tips[name] = &mediaTip{
				Tip:        tip.Tip,
				TipSubtext: tip.TipSubtext,
				TipStyle:   tip.TipStyle,
			}
		}
	}

	return ps
}

func (p *mediaSlot) unmarshalMapFromClient(data dataMap, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys("media_slot",
		"name", "id", "type"); err != nil {
		return err
	}

	p.Name = data.mustGetString("name")
	p.ID = data.mustGetString("id")
	p.Required = data.mustGetBool("required")
	p.clientPlatform = dataSource.clientPlatform()
	p.Type = data.mustGetString("type")
	clientData, err := data.dataMapForKey("client_data")
	if err != nil {
		return err
	} else if clientData == nil {
		return nil
	}

	p.Tip = clientData.mustGetString("tip")
	p.TipSubtext = clientData.mustGetString("tip_subtext")
	p.TipStyle = clientData.mustGetString("tip_style")
	p.OverlayImageLink = clientData.mustGetString("overlay_image_url")
	p.MediaMissingErrorMessage = clientData.mustGetString("media_missing_error_message")
	if p.MediaMissingErrorMessage == "" {
		p.MediaMissingErrorMessage = "Please take all required photos to continue."
	}
	p.InitialCameraDirection = clientData.mustGetString("initial_camera_direction")
	p.FlashState = clientData.mustGetString("flash")

	if clientData.exists("tips") {
		p.Tips = make(map[string]*mediaTip, 1)
		tips, err := clientData.dataMapForKey("tips")
		if err != nil {
			return err
		}

		inlineTips, err := tips.dataMapForKey("inline")
		if err != nil {
			return err
		}

		if inlineTips != nil {
			p.Tips["inline"] = &mediaTip{
				Tip:        inlineTips.mustGetString("tip"),
				TipStyle:   inlineTips.mustGetString("tip_style"),
				TipSubtext: inlineTips.mustGetString("tip_subtext"),
			}
		}

	}

	return nil
}

type mediaQuestion struct {
	*questionInfo
	AllowsMultipleSections        bool         `json:"allows_multiple_sections"`
	AllowsUserDefinedSectionTitle bool         `json:"allows_user_defined_section_title"`
	DisableLastSlotDuplication    bool         `json:"disable_last_slot_duplication"`
	Slots                         []*mediaSlot `json:"media_slots"`

	answer *mediaSectionAnswer
}

func (p *mediaQuestion) staticInfoCopy(context map[string]string) interface{} {
	pCopy := &mediaQuestion{
		questionInfo:                  p.questionInfo.staticInfoCopy(context).(*questionInfo),
		AllowsMultipleSections:        p.AllowsMultipleSections,
		AllowsUserDefinedSectionTitle: p.AllowsUserDefinedSectionTitle,
		DisableLastSlotDuplication:    p.DisableLastSlotDuplication,
		Slots: make([]*mediaSlot, len(p.Slots)),
	}

	for i, slot := range p.Slots {
		pCopy.Slots[i] = slot.staticInfoCopy(context).(*mediaSlot)
	}

	return pCopy
}

func (q *mediaQuestion) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys(questionTypeMedia.String(), "media_slots"); err != nil {
		return err
	}

	var err error
	q.questionInfo, err = populateQuestionInfo(data, parent, questionTypeMedia.String())
	if err != nil {
		return err
	}

	clientData, err := data.dataMapForKey("additional_fields")
	if err != nil {
		return err
	} else if clientData != nil {
		q.AllowsMultipleSections = clientData.mustGetBool("allows_multiple_sections")
		q.AllowsUserDefinedSectionTitle = clientData.mustGetBool("allows_user_defined_section_title")
		q.DisableLastSlotDuplication = clientData.mustGetBool("disable_last_slot_duplication")
	}

	mediaSlots, err := data.getInterfaceSlice("media_slots")
	if err != nil {
		return err
	}

	q.Slots = make([]*mediaSlot, len(mediaSlots))
	for i, slot := range mediaSlots {
		slotMap, err := getDataMap(slot)
		if err != nil {
			return err
		}

		q.Slots[i] = &mediaSlot{}
		if err := q.Slots[i].unmarshalMapFromClient(slotMap, dataSource); err != nil {
			return err
		}
	}

	answer := dataSource.answerForQuestion(q.id())
	if answer != nil {
		msa, ok := answer.(*mediaSectionAnswer)
		if !ok {
			return fmt.Errorf("expected mediaSectionAnswer but got %T", answer)
		}
		q.answer = msa
	}

	return nil
}

func (q *mediaQuestion) TypeName() string {
	return questionTypeMedia.String()
}

// TODO
func (q *mediaQuestion) validateAnswer(pa patientAnswer) error {
	return nil
}

func (q *mediaQuestion) setPatientAnswer(answer patientAnswer) error {
	pqAnswer, ok := answer.(*mediaSectionAnswer)
	if !ok {
		return fmt.Errorf("Expected photo section answer but got %T for question %s", answer, q.LayoutUnitID)
	}

	if !q.AllowsMultipleSections && len(pqAnswer.Sections) > 1 {
		return fmt.Errorf("only single photo section allowed for question %s", q.LayoutUnitID)
	}

	// validate content of each section
	for _, section := range pqAnswer.Sections {
		if section.Name == "" {
			return fmt.Errorf("Name of section cannot be empty for answer to question %s", q.LayoutUnitID)
		}

		if len(section.Media) == 0 {
			return fmt.Errorf("Cannot have empty section defined for question %s", q.LayoutUnitID)
		}

		for _, media := range section.Media {
			if media.Name == "" {
				return fmt.Errorf("Name of media cannot be empty for answer to question: %s", q.LayoutUnitID)
			}

			if media.ServerMediaID == "" && media.LocalMediaID == "" {
				return fmt.Errorf("Local or server MediaID required for all photos in question %s", q.LayoutUnitID)
			}

			if media.SlotID == "" {
				return fmt.Errorf("SlotID required for all media in question: %s", q.LayoutUnitID)
			}

			if media.SlotID == "" {
				return fmt.Errorf("SlotID required for all media in question: %s", q.LayoutUnitID)
			}

		}
	}

	q.answer = pqAnswer
	return nil
}

func (q *mediaQuestion) patientAnswer() (patientAnswer, error) {
	if q.answer == nil {
		return nil, errNoAnswerExists
	}
	return q.answer, nil
}

func (q *mediaQuestion) canPersistAnswer() bool {
	if q.answer == nil {
		return false
	}

	// go through all photos and ensure that none of the photos
	// are still in the process of being uploaded
	for _, section := range q.answer.Sections {
		for _, media := range section.Media {
			if !media.itemUploaded() {
				return false
			}
		}
	}

	return true
}

func (q *mediaQuestion) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	return q.checkQuestionRequirements(q, q.answer)
}

func (q *mediaQuestion) answerForClient() (interface{}, error) {
	if q.answer == nil {
		return nil, errNoAnswerExists
	}

	return q.answer.transformForClient()
}

func (q *mediaQuestion) transformToProtobuf() (proto.Message, error) {
	qInfo, err := transformQuestionInfoToProtobuf(q.questionInfo)
	if err != nil {
		return nil, err
	}

	mediaQuestionProtoBuf := &intake.MediaSectionQuestion{
		QuestionInfo:               qInfo.(*intake.CommonQuestionInfo),
		MediaSlots:                 make([]*intake.MediaSectionQuestion_MediaSlot, len(q.Slots)),
		AllowsMultipleSections:     proto.Bool(q.AllowsMultipleSections),
		UserDefinedSectionTitle:    proto.Bool(q.AllowsUserDefinedSectionTitle),
		DisableLastSlotDuplication: proto.Bool(q.DisableLastSlotDuplication),
	}

	for i, ps := range q.Slots {
		mediaQuestionProtoBuf.MediaSlots[i] = &intake.MediaSectionQuestion_MediaSlot{
			Id:                   proto.String(ps.ID),
			Name:                 proto.String(ps.Name),
			IsRequired:           proto.Bool(ps.Required),
			MediaMissingErrorMsg: proto.String(ps.MediaMissingErrorMessage),
		}

		switch ps.Type {
		case "image":
			mediaQuestionProtoBuf.MediaSlots[i].Type = intake.MediaSectionQuestion_MediaSlot_IMAGE.Enum()
		case "video":
			mediaQuestionProtoBuf.MediaSlots[i].Type = intake.MediaSectionQuestion_MediaSlot_VIDEO.Enum()
		}

		if ps.clientPlatform == android && ps.Tips["inline"] != nil {
			mediaQuestionProtoBuf.MediaSlots[i].Tip = proto.String(ps.Tips["inline"].Tip)
			mediaQuestionProtoBuf.MediaSlots[i].TipSubtext = proto.String(ps.Tips["inline"].TipSubtext)
		} else {
			mediaQuestionProtoBuf.MediaSlots[i].Tip = proto.String(ps.Tip)
			mediaQuestionProtoBuf.MediaSlots[i].TipSubtext = proto.String(ps.TipSubtext)
		}
	}

	if q.answer != nil {
		pb, err := q.answer.transformToProtobuf()
		if err != nil {
			return nil, err
		}

		mediaQuestionProtoBuf.PatientAnswer = pb.(*intake.MediaSectionPatientAnswer)
	}

	return mediaQuestionProtoBuf, nil
}

func (q *mediaQuestion) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	b.WriteString(indentAtDepth(indent, depth) + q.layoutUnitID() + ": " + q.Type + " | " + q.v.String() + "\n")
	b.WriteString(indentAtDepth(indent, depth) + "Q: " + q.Title)
	if q.Subtitle != "" {
		b.WriteString("\n")
		b.WriteString(indentAtDepth(indent, depth) + q.Subtitle)
	}
	if q.answer != nil {
		b.WriteString(indentAtDepth(indent, depth) + q.answer.stringIndent(indent, depth))
	}

	return b.String()
}

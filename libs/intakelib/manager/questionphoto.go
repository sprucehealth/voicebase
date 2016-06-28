package manager

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type photoTip struct {
	Tip        string `json:"tip"`
	TipSubtext string `json:"tip_subtext"`
	TipStyle   string `json:"tip_style"`
}

type photoSlot struct {
	Name                     string `json:"name"`
	ID                       string `json:"id"`
	Required                 bool   `json:"required"`
	Type                     string `json:"type"`
	OverlayImageLink         string `json:"overlay_image_url"`
	PhotoMissingErrorMessage string `json:"photo_missing_error_message"`
	InitialCameraDirection   string `json:"initial_camera_direction"`
	FlashState               string `json:"flash"`

	clientPlatform platform
	photoTip
	Tips map[string]*photoTip `json:"tips"`
}

func (p *photoSlot) staticInfoCopy(context map[string]string) interface{} {
	ps := &photoSlot{
		Name:     p.Name,
		ID:       p.ID,
		Required: p.Required,
		Type:     p.Type,
		photoTip: photoTip{
			Tip:        p.Tip,
			TipSubtext: p.TipSubtext,
			TipStyle:   p.TipStyle,
		},
		OverlayImageLink:         p.OverlayImageLink,
		PhotoMissingErrorMessage: p.PhotoMissingErrorMessage,
		InitialCameraDirection:   p.InitialCameraDirection,
		FlashState:               p.FlashState,
	}

	if p.Tips != nil {
		ps.Tips = make(map[string]*photoTip, len(p.Tips))
		for name, tip := range p.Tips {
			ps.Tips[name] = &photoTip{
				Tip:        tip.Tip,
				TipSubtext: tip.TipSubtext,
				TipStyle:   tip.TipStyle,
			}
		}
	}

	return ps
}

func (p *photoSlot) unmarshalMapFromClient(data dataMap, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys("photo_slot",
		"name", "id"); err != nil {
		return err
	}

	p.Name = data.mustGetString("name")
	p.ID = data.mustGetString("id")
	p.Required = data.mustGetBool("required")
	p.clientPlatform = dataSource.clientPlatform()

	clientData, err := data.dataMapForKey("client_data")
	if err != nil {
		return err
	} else if clientData == nil {
		return nil
	}

	p.Type = clientData.mustGetString("type")
	p.Tip = clientData.mustGetString("tip")
	p.TipSubtext = clientData.mustGetString("tip_subtext")
	p.TipStyle = clientData.mustGetString("tip_style")
	p.OverlayImageLink = clientData.mustGetString("overlay_image_url")
	p.PhotoMissingErrorMessage = clientData.mustGetString("photo_missing_error_message")
	if p.PhotoMissingErrorMessage == "" {
		p.PhotoMissingErrorMessage = "Please take all required photos to continue."
	}
	p.InitialCameraDirection = clientData.mustGetString("initial_camera_direction")
	p.FlashState = clientData.mustGetString("flash")

	if clientData.exists("tips") {
		p.Tips = make(map[string]*photoTip, 1)
		tips, err := clientData.dataMapForKey("tips")
		if err != nil {
			return err
		}

		inlineTips, err := tips.dataMapForKey("inline")
		if err != nil {
			return err
		}

		if inlineTips != nil {
			p.Tips["inline"] = &photoTip{
				Tip:        inlineTips.mustGetString("tip"),
				TipStyle:   inlineTips.mustGetString("tip_style"),
				TipSubtext: inlineTips.mustGetString("tip_subtext"),
			}
		}

	}

	return nil
}

type photoQuestion struct {
	*questionInfo
	AllowsMultipleSections        bool         `json:"allows_multiple_sections"`
	AllowsUserDefinedSectionTitle bool         `json:"allows_user_defined_section_title"`
	DisableLastSlotDuplication    bool         `json:"disable_last_slot_duplication"`
	Slots                         []*photoSlot `json:"photo_slots"`

	answer *photoSectionAnswer
}

func (p *photoQuestion) staticInfoCopy(context map[string]string) interface{} {
	pCopy := &photoQuestion{
		questionInfo:                  p.questionInfo.staticInfoCopy(context).(*questionInfo),
		AllowsMultipleSections:        p.AllowsMultipleSections,
		AllowsUserDefinedSectionTitle: p.AllowsUserDefinedSectionTitle,
		DisableLastSlotDuplication:    p.DisableLastSlotDuplication,
		Slots: make([]*photoSlot, len(p.Slots)),
	}

	for i, slot := range p.Slots {
		pCopy.Slots[i] = slot.staticInfoCopy(context).(*photoSlot)
	}

	return pCopy
}

func (q *photoQuestion) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys(questionTypePhoto.String(), "photo_slots"); err != nil {
		return err
	}

	var err error
	q.questionInfo, err = populateQuestionInfo(data, parent, questionTypePhoto.String())
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

	photoSlots, err := data.getInterfaceSlice("photo_slots")
	if err != nil {
		return err
	}

	q.Slots = make([]*photoSlot, len(photoSlots))
	for i, slot := range photoSlots {
		slotMap, err := getDataMap(slot)
		if err != nil {
			return err
		}

		q.Slots[i] = &photoSlot{}
		if err := q.Slots[i].unmarshalMapFromClient(slotMap, dataSource); err != nil {
			return err
		}
	}

	if data.exists("answers") {
		q.answer = &photoSectionAnswer{}
		if err := q.answer.unmarshalMapFromClient(data); err != nil {
			return err
		}
	}

	return nil
}

func (q *photoQuestion) TypeName() string {
	return questionTypePhoto.String()
}

// TODO
func (q *photoQuestion) validateAnswer(pa patientAnswer) error {
	return nil
}

func (q *photoQuestion) setPatientAnswer(answer patientAnswer) error {
	pqAnswer, ok := answer.(*photoSectionAnswer)
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

		if len(section.Photos) == 0 {
			return fmt.Errorf("Cannot have empty section defined for question %s", q.LayoutUnitID)
		}

		for _, photo := range section.Photos {
			if photo.Name == "" {
				return fmt.Errorf("Name of photo cannot be empty for answer to question: %s", q.LayoutUnitID)
			}

			if photo.ServerPhotoID == "" && photo.LocalPhotoID == "" {
				return fmt.Errorf("Local or server PhotoID required for all photos in question %s", q.LayoutUnitID)
			}

			if photo.SlotID == "" {
				return fmt.Errorf("SlotID required for all photos in question: %s", q.LayoutUnitID)
			}

			if photo.SlotID == "" {
				return fmt.Errorf("SlotID required for all photos in question: %s", q.LayoutUnitID)
			}

		}
	}

	q.answer = pqAnswer
	return nil
}

func (q *photoQuestion) patientAnswer() (patientAnswer, error) {
	if q.answer == nil {
		return nil, errNoAnswerExists
	}
	return q.answer, nil
}

func (q *photoQuestion) canPersistAnswer() bool {
	if q.answer == nil {
		return false
	}

	// go through all photos and ensure that none of the photos
	// are still in the process of being uploaded
	for _, section := range q.answer.Sections {
		for _, photo := range section.Photos {
			if !photo.itemUploaded() {
				return false
			}
		}
	}

	return true
}

func (q *photoQuestion) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	return q.checkQuestionRequirements(q, q.answer)
}

func (q *photoQuestion) marshalAnswerForClient() ([]byte, error) {
	if q.answer == nil {
		return nil, errNoAnswerExists
	}

	if q.visibility() == hidden {
		return q.answer.marshalEmptyJSONForClient()
	}

	return q.answer.marshalJSONForClient()
}

func (q *photoQuestion) transformToProtobuf() (proto.Message, error) {
	qInfo, err := transformQuestionInfoToProtobuf(q.questionInfo)
	if err != nil {
		return nil, err
	}

	photoQuestionProtoBuf := &intake.PhotoSectionQuestion{
		QuestionInfo:               qInfo.(*intake.CommonQuestionInfo),
		PhotoSlots:                 make([]*intake.PhotoSectionQuestion_PhotoSlot, len(q.Slots)),
		AllowsMultipleSections:     proto.Bool(q.AllowsMultipleSections),
		UserDefinedSectionTitle:    proto.Bool(q.AllowsUserDefinedSectionTitle),
		DisableLastSlotDuplication: proto.Bool(q.DisableLastSlotDuplication),
	}

	for i, ps := range q.Slots {
		photoQuestionProtoBuf.PhotoSlots[i] = &intake.PhotoSectionQuestion_PhotoSlot{
			Id:                   proto.String(ps.ID),
			Name:                 proto.String(ps.Name),
			IsRequired:           proto.Bool(ps.Required),
			PhotoMissingErrorMsg: proto.String(ps.PhotoMissingErrorMessage),
		}

		if ps.clientPlatform == android && ps.Tips["inline"] != nil {
			photoQuestionProtoBuf.PhotoSlots[i].Tip = proto.String(ps.Tips["inline"].Tip)
			photoQuestionProtoBuf.PhotoSlots[i].TipSubtext = proto.String(ps.Tips["inline"].TipSubtext)
		} else {
			photoQuestionProtoBuf.PhotoSlots[i].Tip = proto.String(ps.Tip)
			photoQuestionProtoBuf.PhotoSlots[i].TipSubtext = proto.String(ps.TipSubtext)
		}
	}

	if q.answer != nil {
		pb, err := q.answer.transformToProtobuf()
		if err != nil {
			return nil, err
		}

		photoQuestionProtoBuf.PatientAnswer = pb.(*intake.PhotoSectionPatientAnswer)
	}

	return photoQuestionProtoBuf, nil
}

func (q *photoQuestion) stringIndent(indent string, depth int) string {
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

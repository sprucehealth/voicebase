package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type photoSlotAnswerItem struct {
	Name   string `json:"name"`
	URL    string `json:"link,omitempty"`
	SlotID string `json:"slot_id"`

	// LocalPhotoID is set by the client when the photo is in the process of
	// being uploaded.
	LocalPhotoID string `json:"local_photo_id,omitempty"`

	// ServerPhotoID represents the ID of an uploaded photo.
	ServerPhotoID string `json:"server_photo_id,omitempty"`
}

func (p *photoSlotAnswerItem) replaceID(idReplacementData interface{}) error {
	pir, ok := idReplacementData.(*photoIDReplacement)
	if !ok {
		return fmt.Errorf("Unexpected type: %T", idReplacementData)
	}
	p.LocalPhotoID = ""
	p.ServerPhotoID = pir.ID
	if pir.URL != "" {
		p.URL = pir.URL
	}
	return nil
}

func (p *photoSlotAnswerItem) itemUploaded() bool {
	return p.LocalPhotoID == "" && p.ServerPhotoID != ""
}

func (p *photoSlotAnswerItem) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys(
		"photo_slot_answer_item",
		"name", "photo_url", "photo_id", "slot_id"); err != nil {
		return err
	}

	p.Name = data.mustGetString("name")
	p.URL = data.mustGetString("photo_url")
	p.SlotID = data.mustGetString("slot_id")
	p.ServerPhotoID = data.mustGetString("photo_id")

	return nil
}

type photoSectionAnswerItem struct {
	Name   string                 `json:"name"`
	Photos []*photoSlotAnswerItem `json:"photos"`
}

func (p *photoSectionAnswerItem) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys(
		"photo_section_answer_item",
		"name", "photos"); err != nil {
		return err
	}

	p.Name = data.mustGetString("name")
	slots, err := data.getInterfaceSlice("photos")
	if err != nil {
		return err
	}

	p.Photos = make([]*photoSlotAnswerItem, len(slots))
	for i, slot := range slots {

		slotMap, err := getDataMap(slot)
		if err != nil {
			return err
		}

		var s photoSlotAnswerItem
		if err := s.unmarshalMapFromClient(slotMap); err != nil {
			return err
		}

		p.Photos[i] = &s
	}

	return nil
}

type photoSectionAnswer struct {
	QuestionID string                    `json:"question_id"`
	Sections   []*photoSectionAnswerItem `json:"sections"`
}

func (m *photoSectionAnswer) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	for _, sItem := range m.Sections {
		b.WriteString("\n")
		b.WriteString(indentAtDepth(indent, depth) + "A: " + sItem.Name)
		for _, pItem := range sItem.Photos {
			b.WriteString("\n")
			b.WriteString(indentAtDepth(indent, depth+1) + "Photo: " + pItem.Name)
		}
	}
	return b.String()
}

func (m *photoSectionAnswer) setQuestionID(questionID string) {
	m.QuestionID = questionID
}

func (m *photoSectionAnswer) questionID() string {
	return m.QuestionID
}

func (m *photoSectionAnswer) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("photo_section_answer", "answers"); err != nil {
		return err
	}

	answers, err := data.getInterfaceSlice("answers")
	if err != nil {
		return err
	} else if len(answers) == 0 {
		return errors.New("No answers found when trying to parse photo section answer.")
	}

	m.Sections = make([]*photoSectionAnswerItem, len(answers))
	for i, photoSection := range answers {
		photoSectionMap, err := getDataMap(photoSection)
		if err != nil {
			return err
		}

		var ps photoSectionAnswerItem
		if err := ps.unmarshalMapFromClient(photoSectionMap); err != nil {
			return err
		}

		m.Sections[i] = &ps
	}

	return nil
}

func (m *photoSectionAnswer) unmarshalProtobuf(data []byte) error {
	var pb intake.PhotoSectionPatientAnswer
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}

	m.Sections = make([]*photoSectionAnswerItem, len(pb.Entries))
	for i, section := range pb.Entries {
		if section.Name == nil {
			return errors.New("section name missing")
		}

		m.Sections[i] = &photoSectionAnswerItem{
			Name:   *section.Name,
			Photos: make([]*photoSlotAnswerItem, len(section.Photos)),
		}

		for j, slot := range section.Photos {

			if slot.Name == nil {
				return errors.New("photo slot name missing")
			} else if slot.SlotId == nil {
				return errors.New("photo slot id missing")
			} else if slot.Id == nil {
				return errors.New("photo id missing")
			} else if slot.Id.Id == nil {
				return errors.New("photo id missing")
			}

			m.Sections[i].Photos[j] = &photoSlotAnswerItem{
				Name:   *slot.Name,
				SlotID: *slot.SlotId,
			}

			if *slot.Id.Type == intake.ID_LOCAL {
				m.Sections[i].Photos[j].LocalPhotoID = *slot.Id.Id
			} else {
				m.Sections[i].Photos[j].ServerPhotoID = *slot.Id.Id
			}

			if slot.Link != nil {
				m.Sections[i].Photos[j].URL = *slot.Link
			}

		}
	}

	return nil
}

func (m *photoSectionAnswer) transformToProtobuf() (proto.Message, error) {
	var pb intake.PhotoSectionPatientAnswer
	pb.Entries = make([]*intake.PhotoSectionPatientAnswer_PhotoSectionEntry, len(m.Sections))

	for i, section := range m.Sections {
		pb.Entries[i] = &intake.PhotoSectionPatientAnswer_PhotoSectionEntry{
			Name:   proto.String(section.Name),
			Photos: make([]*intake.PhotoSectionPatientAnswer_PhotoSectionEntry_PhotoSlotAnswer, len(section.Photos)),
		}

		for j, slot := range section.Photos {
			var idType intake.ID_Type
			var id string
			if slot.LocalPhotoID != "" {
				idType = intake.ID_LOCAL
				id = slot.LocalPhotoID
			} else if slot.ServerPhotoID != "" {
				idType = intake.ID_SERVER
				id = slot.ServerPhotoID
			} else {
				return nil, fmt.Errorf("Neither local nor server photo id set")
			}
			pb.Entries[i].Photos[j] = &intake.PhotoSectionPatientAnswer_PhotoSectionEntry_PhotoSlotAnswer{
				Name: proto.String(slot.Name),
				Id: &intake.ID{
					Id:   proto.String(id),
					Type: idType.Enum(),
				},
				SlotId: proto.String(slot.SlotID),
				Link:   proto.String(slot.URL),
			}
		}
	}

	return &pb, nil
}

type photoSectionClientJSONItem struct {
	Name    string `json:"name"`
	SlotID  string `json:"slot_id"`
	PhotoID string `json:"photo_id"`
}

type photoSectionClientJSON struct {
	Name   string                        `json:"name"`
	Photos []*photoSectionClientJSONItem `json:"photos"`
}

type photoSectionListClientJSON struct {
	QuestionID string                    `json:"question_id"`
	Sections   []*photoSectionClientJSON `json:"answered_photo_sections"`
}

func (m *photoSectionAnswer) marshalEmptyJSONForClient() ([]byte, error) {
	return json.Marshal(photoSectionListClientJSON{
		QuestionID: sanitizeQuestionID(m.QuestionID),
		Sections:   []*photoSectionClientJSON{},
	})
}

func (m *photoSectionAnswer) marshalJSONForClient() ([]byte, error) {
	clientJSON := &photoSectionListClientJSON{
		QuestionID: sanitizeQuestionID(m.QuestionID),
		Sections:   make([]*photoSectionClientJSON, len(m.Sections)),
	}

	for i, section := range m.Sections {
		clientJSON.Sections[i] = &photoSectionClientJSON{
			Name:   section.Name,
			Photos: make([]*photoSectionClientJSONItem, len(section.Photos)),
		}

		for j, slot := range section.Photos {
			clientJSON.Sections[i].Photos[j] = &photoSectionClientJSONItem{
				Name:    slot.Name,
				SlotID:  slot.SlotID,
				PhotoID: slot.ServerPhotoID,
			}
		}
	}

	return json.Marshal(clientJSON)
}

func (m *photoSectionAnswer) equals(other patientAnswer) bool {
	if m == nil && other == nil {
		return true
	} else if m == nil || other == nil {
		return false
	}

	otherPSA, ok := other.(*photoSectionAnswer)
	if !ok {
		return false
	}

	if len(m.Sections) != len(otherPSA.Sections) {
		return false
	}

	for i, section := range m.Sections {
		if section.Name != otherPSA.Sections[i].Name {
			return false
		}

		if len(section.Photos) != len(otherPSA.Sections[i].Photos) {
			return false
		}

		for j, photo := range section.Photos {
			if photo.Name != otherPSA.Sections[i].Photos[j].Name {
				return false
			} else if photo.SlotID != otherPSA.Sections[i].Photos[j].SlotID {
				return false
			} else if photo.LocalPhotoID != otherPSA.Sections[i].Photos[j].LocalPhotoID {
				return false
			} else if photo.ServerPhotoID != otherPSA.Sections[i].Photos[j].ServerPhotoID {
				return false
			}
		}
	}

	return true
}

func (m *photoSectionAnswer) itemsToBeUploaded() map[string]uploadableItem {
	photoMap := make(map[string]uploadableItem)
	for _, section := range m.Sections {
		for _, photo := range section.Photos {
			if !photo.itemUploaded() {
				photoMap[photo.LocalPhotoID] = photo
			}
		}
	}

	return photoMap
}

func (m *photoSectionAnswer) isEmpty() bool {
	return len(m.Sections) == 0
}

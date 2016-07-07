package manager

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type mediaSlotAnswerItem struct {
	Name         string `json:"name"`
	URL          string `json:"url,omitempty"`
	SlotID       string `json:"slot_id"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Type         string `json:"type"`

	// LocalMediaID is set by the client when the photo is in the process of
	// being uploaded.
	LocalMediaID string `json:"local_media_id,omitempty"`

	// ServerMediaID represents the ID of an uploaded photo.
	ServerMediaID string `json:"server_media_id,omitempty"`
}

func (p *mediaSlotAnswerItem) replaceID(idReplacementData interface{}) error {
	pir, ok := idReplacementData.(*mediaIDReplacement)
	if !ok {
		return fmt.Errorf("Unexpected type: %T", idReplacementData)
	}
	p.LocalMediaID = ""
	p.ServerMediaID = pir.ID
	p.URL = pir.URL
	p.ThumbnailURL = pir.ThumbnailURL

	return nil
}

func (p *mediaSlotAnswerItem) itemUploaded() bool {
	return p.LocalMediaID == "" && p.ServerMediaID != ""
}

func (p *mediaSlotAnswerItem) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys(
		"photo_slot_answer_item",
		"name", "url", "thumbnail_url", "media_id", "slot_id", "type"); err != nil {
		return err
	}

	p.Name = data.mustGetString("name")
	p.URL = data.mustGetString("url")
	p.ThumbnailURL = data.mustGetString("thumbnail_url")
	p.Type = data.mustGetString("type")
	p.SlotID = data.mustGetString("slot_id")
	p.ServerMediaID = data.mustGetString("media_id")

	return nil
}

type mediaSectionAnswerItem struct {
	Name  string                 `json:"name"`
	Media []*mediaSlotAnswerItem `json:"media"`
}

func (p *mediaSectionAnswerItem) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys(
		"photo_section_answer_item",
		"name", "media"); err != nil {
		return errors.Trace(err)

	}

	p.Name = data.mustGetString("name")
	slots, err := data.getInterfaceSlice("media")
	if err != nil {
		return errors.Trace(err)
	}

	p.Media = make([]*mediaSlotAnswerItem, len(slots))
	for i, slot := range slots {

		slotMap, err := getDataMap(slot)
		if err != nil {
			return errors.Trace(err)
		}

		var s mediaSlotAnswerItem
		if err := s.unmarshalMapFromClient(slotMap); err != nil {
			return errors.Trace(err)
		}

		p.Media[i] = &s
	}

	return nil
}

type mediaSectionAnswer struct {
	Sections []*mediaSectionAnswerItem `json:"sections"`
}

func (m *mediaSectionAnswer) stringIndent(indent string, depth int) string {
	var b bytes.Buffer
	for _, sItem := range m.Sections {
		b.WriteString("\n")
		b.WriteString(indentAtDepth(indent, depth) + "A: " + sItem.Name)
		for _, pItem := range sItem.Media {
			b.WriteString("\n")
			b.WriteString(indentAtDepth(indent, depth+1) + "Media: " + pItem.Name)
		}
	}
	return b.String()
}

func (m *mediaSectionAnswer) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("photo_section_answer", "sections"); err != nil {
		return err
	}

	answers, err := data.getInterfaceSlice("sections")
	if err != nil {
		return err
	} else if len(answers) == 0 {
		return errors.New("No answers found when trying to parse photo section answer.")
	}

	m.Sections = make([]*mediaSectionAnswerItem, len(answers))
	for i, mediaSection := range answers {
		mediaSectionMap, err := getDataMap(mediaSection)
		if err != nil {
			return err
		}

		var ps mediaSectionAnswerItem
		if err := ps.unmarshalMapFromClient(mediaSectionMap); err != nil {
			return err
		}

		m.Sections[i] = &ps
	}

	return nil
}

func (m *mediaSectionAnswer) unmarshalProtobuf(data []byte) error {
	var pb intake.MediaSectionPatientAnswer
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}

	m.Sections = make([]*mediaSectionAnswerItem, len(pb.Entries))
	for i, section := range pb.Entries {
		if section.Name == nil {
			return errors.New("section name missing")
		}

		m.Sections[i] = &mediaSectionAnswerItem{
			Name:  *section.Name,
			Media: make([]*mediaSlotAnswerItem, len(section.Media)),
		}

		for j, slot := range section.Media {

			if slot.Name == nil {
				return errors.New("media slot name missing")
			} else if slot.SlotId == nil {
				return errors.New("media slot id missing")
			} else if slot.Id == nil {
				return errors.New("media id missing")
			} else if slot.Id.Id == nil {
				return errors.New("media id missing")
			}

			m.Sections[i].Media[j] = &mediaSlotAnswerItem{
				Name:   *slot.Name,
				SlotID: *slot.SlotId,
			}

			if *slot.Id.Type == intake.ID_LOCAL {
				m.Sections[i].Media[j].LocalMediaID = *slot.Id.Id
			} else {
				m.Sections[i].Media[j].ServerMediaID = *slot.Id.Id
			}

			if slot.Link != nil {
				m.Sections[i].Media[j].URL = *slot.Link
			}

		}
	}

	return nil
}

func (m *mediaSectionAnswer) transformToProtobuf() (proto.Message, error) {
	var pb intake.MediaSectionPatientAnswer
	pb.Entries = make([]*intake.MediaSectionPatientAnswer_MediaSectionEntry, len(m.Sections))

	for i, section := range m.Sections {
		pb.Entries[i] = &intake.MediaSectionPatientAnswer_MediaSectionEntry{
			Name:  proto.String(section.Name),
			Media: make([]*intake.MediaSectionPatientAnswer_MediaSectionEntry_MediaSlotAnswer, len(section.Media)),
		}

		for j, slot := range section.Media {
			var idType intake.ID_Type
			var id string
			if slot.LocalMediaID != "" {
				idType = intake.ID_LOCAL
				id = slot.LocalMediaID
			} else if slot.ServerMediaID != "" {
				idType = intake.ID_SERVER
				id = slot.ServerMediaID
			} else {
				return nil, fmt.Errorf("Neither local nor server media id set")
			}
			pb.Entries[i].Media[j] = &intake.MediaSectionPatientAnswer_MediaSectionEntry_MediaSlotAnswer{
				Name: proto.String(slot.Name),
				Id: &intake.ID{
					Id:   proto.String(id),
					Type: idType.Enum(),
				},
				SlotId:        proto.String(slot.SlotID),
				Link:          proto.String(slot.URL),
				ThumbnailLink: proto.String(slot.ThumbnailURL),
			}
		}
	}

	return &pb, nil
}

type mediaSectionClientJSONItem struct {
	Name    string `json:"name"`
	SlotID  string `json:"slot_id"`
	MediaID string `json:"media_id"`
	Type    string `json:"type"`
}

type mediaSectionClientJSON struct {
	Name  string                        `json:"name"`
	Media []*mediaSectionClientJSONItem `json:"media"`
}

type mediaSectionListClientJSON struct {
	Type     string                    `json:"type"`
	Sections []*mediaSectionClientJSON `json:"sections"`
}

func (m *mediaSectionAnswer) transformForClient() (interface{}, error) {
	clientJSON := &mediaSectionListClientJSON{
		Type:     questionTypeMedia.String(),
		Sections: make([]*mediaSectionClientJSON, len(m.Sections)),
	}

	for i, section := range m.Sections {
		clientJSON.Sections[i] = &mediaSectionClientJSON{
			Name:  section.Name,
			Media: make([]*mediaSectionClientJSONItem, len(section.Media)),
		}

		for j, slot := range section.Media {
			clientJSON.Sections[i].Media[j] = &mediaSectionClientJSONItem{
				Name:    slot.Name,
				SlotID:  slot.SlotID,
				MediaID: slot.ServerMediaID,
				Type:    slot.Type,
			}
		}
	}

	return clientJSON, nil
}

func (m *mediaSectionAnswer) equals(other patientAnswer) bool {
	if m == nil && other == nil {
		return true
	} else if m == nil || other == nil {
		return false
	}

	otherPSA, ok := other.(*mediaSectionAnswer)
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

		if len(section.Media) != len(otherPSA.Sections[i].Media) {
			return false
		}

		for j, media := range section.Media {
			if media.Name != otherPSA.Sections[i].Media[j].Name {
				return false
			} else if media.SlotID != otherPSA.Sections[i].Media[j].SlotID {
				return false
			} else if media.LocalMediaID != otherPSA.Sections[i].Media[j].LocalMediaID {
				return false
			} else if media.ServerMediaID != otherPSA.Sections[i].Media[j].ServerMediaID {
				return false
			}
		}
	}

	return true
}

func (m *mediaSectionAnswer) itemsToBeUploaded() map[string]uploadableItem {
	mediaMap := make(map[string]uploadableItem)
	for _, section := range m.Sections {
		for _, media := range section.Media {
			if !media.itemUploaded() {
				mediaMap[media.LocalMediaID] = media
			}
		}
	}

	return mediaMap
}

func (m *mediaSectionAnswer) isEmpty() bool {
	return len(m.Sections) == 0
}

package messages

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/golog"
	"fmt"
	"time"
)

type message struct {
	Type        string        `json:"type"`
	Time        time.Time     `json:"date_time"`
	SenderId    int64         `json:"sender_participant_id"`
	Message     string        `json:"message"`
	Attachments []*attachment `json:"attachments,omitempty"`
}

type attachment struct {
	Type            string `json:"type"`
	URL             string `json:"url,omitempty"`
	TreatmentPlanId int64  `json:"treatment_plan_id,omitempty"`
}

type conversationListItem struct {
	Id                int64     `json:"id"`
	Title             string    `json:"title"`
	LastMessageTime   time.Time `json:"last_message_date_time"`
	LastParticipantId int64     `json:"last_message_participant_id,string"`
	MessageCount      int       `json:"message_count"`
	Unread            bool      `json:"unread"`
}

type participant struct {
	Id           int64  `json:"participant_id,string"`
	Name         string `json:"name"`
	Subtitle     string `json:"subtitle,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Initials     string `json:"initials"`
}

type conversationListResponse struct {
	Conversations []*conversationListItem `json:"conversations"`
	Participants  []*participant          `json:"participants"`
}

type conversationResponse struct {
	Id           int64          `json:"conversation_id"`
	Title        string         `json:"title"`
	Items        []*message     `json:"items"`
	Participants []*participant `json:"participants"`
}

type attachments struct {
	Photos []int64 `json:"photos"`
}

type newConversationRequest struct {
	PatientId   int64        `json:"patient_id"`
	TopicId     int64        `json:"topic_id"`
	Message     string       `json:"message"`
	Attachments *attachments `json:"attachments"`
}

type replyRequest struct {
	ConversationId int64        `json:"conversation_id"`
	Message        string       `json:"message"`
	Attachments    *attachments `json:"attachments"`
}

type conversationRequest struct {
	ConversationId int64 `schema:"conversation_id,required"`
}

func conversationsToConversationList(con []*common.Conversation, personId int64) []*conversationListItem {
	items := make([]*conversationListItem, len(con))
	for i, c := range con {
		item := &conversationListItem{
			Id:                c.Id,
			Title:             c.Title,
			LastMessageTime:   c.LastMessageTime,
			LastParticipantId: c.LastParticipantId,
			MessageCount:      c.MessageCount,
			Unread:            c.Unread && c.OwnerId == personId,
		}
		items[i] = item
	}
	return items
}

func peopleToParticipants(people map[int64]*common.Person) []*participant {
	parts := make([]*participant, 0, len(people))
	for _, per := range people {
		p := &participant{
			Id: per.Id,
		}
		switch per.RoleType {
		case api.PATIENT_ROLE:
			p.Name = fmt.Sprintf("%s %s", per.Patient.FirstName, per.Patient.LastName)
			if len(per.Patient.FirstName) > 0 {
				p.Initials += per.Patient.FirstName[:1]
			}
			if len(per.Patient.LastName) > 0 {
				p.Initials += per.Patient.LastName[:1]
			}
		case api.DOCTOR_ROLE:
			p.Name = fmt.Sprintf("%s %s", per.Doctor.FirstName, per.Doctor.LastName)
			if len(per.Doctor.FirstName) > 0 {
				p.Initials += per.Doctor.FirstName[:1]
			}
			if len(per.Doctor.LastName) > 0 {
				p.Initials += per.Doctor.LastName[:1]
			}
			p.ThumbnailURL = fmt.Sprintf("spruce:///image/thumbnail_care_team_%d", per.RoleId)
			p.Subtitle = "Dermatologist" // TODO
		}
		parts = append(parts, p)
	}
	return parts
}

func messageList(msgs []*common.ConversationMessage) []*message {
	mr := make([]*message, len(msgs))
	for i, m := range msgs {
		mr[i] = &message{
			Type:        "conversation_item:message",
			Time:        m.Time,
			SenderId:    m.FromId,
			Message:     m.Body,
			Attachments: make([]*attachment, len(m.Attachments)),
		}
		for j, a := range m.Attachments {
			switch a.ItemType {
			case common.AttachmentTypePhoto:
				mr[i].Attachments[j] = &attachment{
					Type: "attachment:photo",
					URL:  "TODO", // TODO
				}
			default:
				golog.Errorf("Unknown attachment type %s for message %d", a.ItemType, m.Id)
				continue
			}
		}
	}
	return mr
}

func isPersonAParticipant(dataAPI api.DataAPI, conversationId, personId int64) (bool, error) {
	pars, err := dataAPI.GetConversationParticipantIds(conversationId)
	if err != nil {
		return false, err
	}
	for _, id := range pars {
		if id == personId {
			return true, nil
		}
	}
	return false, nil
}

func parseAttachments(dataAPI api.DataAPI, att *attachments, personId int64) ([]*common.ConversationAttachment, error) {
	var attachments []*common.ConversationAttachment
	if att != nil {
		for _, photoID := range att.Photos {
			photo, err := dataAPI.GetPhoto(photoID)
			if err != nil {
				return nil, err
			}
			if photo.UploaderId != personId {
				return nil, api.NoRowsError
			}
			attachments = append(attachments, &common.ConversationAttachment{
				ItemType: common.AttachmentTypePhoto,
				ItemId:   photoID,
			})
		}
	}
	return attachments, nil
}

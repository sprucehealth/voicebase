package messages

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_url"
	"carefront/common"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/schema"
)

type message struct {
	Type        string        `json:"type"`
	Time        time.Time     `json:"date_time"`
	SenderId    int64         `json:"sender_participant_id,string"`
	Message     string        `json:"message"`
	Attachments []*attachment `json:"attachments,omitempty"`
}

type attachment struct {
	Type            string `json:"type"`
	URL             string `json:"url,omitempty"`
	TreatmentPlanId int64  `json:"treatment_plan_id,string,omitempty"`
}

type ConversationListItem struct {
	Id                int64     `json:"id,string"`
	Title             string    `json:"title"`
	LastMessageTime   time.Time `json:"last_message_date_time"`
	LastParticipantId int64     `json:"last_message_participant_id,string"`
	MessageCount      int       `json:"message_count"`
	Unread            bool      `json:"unread"`
}

type Participant struct {
	Id           int64                `json:"participant_id,string"`
	Name         string               `json:"name"`
	Subtitle     string               `json:"subtitle,omitempty"`
	ThumbnailURL *app_url.SpruceAsset `json:"thumbnail_url,omitempty"`
	Initials     string               `json:"initials"`
}

type ConversationListResponse struct {
	Conversations []*ConversationListItem `json:"conversations"`
	Participants  []*Participant          `json:"participants"`
}

type conversationResponse struct {
	Id           int64          `json:"conversation_id,string"`
	Title        string         `json:"title"`
	Items        []*message     `json:"items"`
	Participants []*Participant `json:"participants"`
}

type attachments struct {
	Photos []string `json:"photos"`
}

type NewConversationRequest struct {
	PatientId   int64        `json:"patient_id,string"`
	TopicId     int64        `json:"topic_id,string"`
	Message     string       `json:"message"`
	Attachments *attachments `json:"attachments"`
}

type NewConversationResponse struct {
	ConversationId int64 `json:"conversation_id,string"`
}

type ReplyRequest struct {
	ConversationId int64        `json:"conversation_id,string"`
	Message        string       `json:"message"`
	Attachments    *attachments `json:"attachments"`
}

type conversationRequest struct {
	ConversationId int64 `schema:"conversation_id,required"`
}

func conversationsToConversationList(con []*common.Conversation, personId int64) []*ConversationListItem {
	items := make([]*ConversationListItem, len(con))
	for i, c := range con {
		item := &ConversationListItem{
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

func peopleToParticipants(people map[int64]*common.Person) []*Participant {
	parts := make([]*Participant, 0, len(people))
	for _, per := range people {
		p := &Participant{
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
			p.ThumbnailURL = per.Doctor.SmallThumbnailUrl
			p.Subtitle = "Dermatologist"
		}
		parts = append(parts, p)
	}
	return parts
}

func messageList(msgs []*common.ConversationMessage, req *http.Request) []*message {
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
					URL:  apiservice.CreatePhotoUrl(a.ItemId, m.Id, common.ClaimerTypeConversationMessage, req.Host),
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
		for _, photoIDStr := range att.Photos {
			photoID, err := strconv.ParseInt(photoIDStr, 10, 64)
			if err != nil {
				return nil, err
			}
			photo, err := dataAPI.GetPhoto(photoID)
			if err != nil {
				return nil, err
			}
			if photo.UploaderId != personId || photo.ClaimerType != "" {
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

func markConversationAsRead(w http.ResponseWriter, r *http.Request, dataAPI api.DataAPI, personId int64) {
	if r.Method != apiservice.HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var req conversationRequest
	if err := schema.NewDecoder().Decode(&req, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	con, err := dataAPI.GetConversation(req.ConversationId)
	if err == api.NoRowsError {
		http.NotFound(w, r)
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get conversation: "+err.Error())
		return
	}

	// Make sure only the current owner can mark the conversation as read
	if con.OwnerId != personId {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Only the current owner of the conversation can flag it as read")
		return
	}
	if err := dataAPI.MarkConversationAsRead(req.ConversationId); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to mark conversation as read: "+err.Error())
		return
	}

	dispatch.Default.PublishAsync(&ConversationReadEvent{
		ConversationId: req.ConversationId,
		FromId:         personId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

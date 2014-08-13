package messages

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/storage"
)

type Participant struct {
	ID           int64                `json:"participant_id,string"`
	Name         string               `json:"name"`
	Initials     string               `json:"initials"`
	Subtitle     string               `json:"subtitle,omitempty"`
	ThumbnailURL *app_url.SpruceAsset `json:"thumbnail_url,omitempty"`
}

type Message struct {
	ID          int64         `json:"message_id,string"`
	Type        string        `json:"type"`
	Time        time.Time     `json:"date_time"`
	SenderID    int64         `json:"sender_participant_id,string"`
	Message     string        `json:"message"`
	Attachments []*Attachment `json:"attachments,omitempty"`
	StatusText  string        `json:"status_text,omitempty"`
}

type ListResponse struct {
	Items        []*Message     `json:"items"`
	Participants []*Participant `json:"participants"`
}

type listHandler struct {
	dataAPI api.DataAPI
	store   storage.Store
}

func NewListHandler(dataAPI api.DataAPI, store storage.Store) http.Handler {
	return &listHandler{dataAPI: dataAPI, store: store}
}

func (h *listHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	ctxt := apiservice.GetContext(r)

	caseID, err := strconv.ParseInt(r.FormValue("case_id"), 10, 64)
	if err != nil {
		return false, apiservice.NewValidationError("bad case_id", r)
	}

	cas, err := h.dataAPI.GetPatientCaseFromId(caseID)
	if err == api.NoRowsError {
		return false, apiservice.NewResourceNotFoundError("Case not found", r)
	}
	ctxt.RequestCache[apiservice.PatientCase] = cas

	_, _, err = validateAccess(h.dataAPI, r, cas)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (h *listHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	cas := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)

	msgs, err := h.dataAPI.ListCaseMessages(cas.Id.Int64(), ctxt.Role)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	participants, err := h.dataAPI.CaseMessageParticipants(cas.Id.Int64(), true)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &ListResponse{}
	for _, msg := range msgs {

		msgType := "conversation_item:message"
		if msg.IsPrivate {
			msgType = "conversation_item:private_message"
		}

		m := &Message{
			ID:         msg.ID,
			Type:       msgType,
			Time:       msg.Time,
			SenderID:   msg.PersonID,
			Message:    msg.Body,
			StatusText: msg.EventText,
		}

		for _, att := range msg.Attachments {
			a := &Attachment{
				ID: att.ItemID,
			}

			switch att.ItemType {
			case common.AttachmentTypePhoto:
				a.Type = "attachment:" + att.ItemType
				a.URL = apiservice.CreatePhotoUrl(att.ItemID, msg.ID, common.ClaimerTypeConversationMessage, r.Host)
			case common.AttachmentTypeTreatmentPlan:
				a.Type = "attachment:" + att.ItemType
				a.URL = app_url.ViewTreatmentPlanAction(att.ItemID).String()
			case common.AttachmentTypeMedia:
				mediaType := strings.Split(att.MimeType, "/")
				switch mediaType[0] {
				case "image":
					a.Type = "attachment:photo"
				case "audio":
					a.Type = "attachment:audio"
				}
				a.MimeType = att.MimeType
				media, err := h.dataAPI.GetMedia(att.ItemID)

				if err == api.NoRowsError {
					http.NotFound(w, r)
					return
				} else if err != nil {
					apiservice.WriteError(w, http.StatusInternalServerError, "Failed to get media: "+err.Error())
					return
				}

				if media.ClaimerID != msg.ID {
					http.NotFound(w, r)
					return
				}
				newURL, err := h.store.GetSignedURL(media.URL)

				if err != nil {
					apiservice.WriteError(w, http.StatusInternalServerError, "Failed to get media: "+err.Error())
					return
				}
				a.URL = newURL
			}

			m.Attachments = append(m.Attachments, a)
		}

		res.Items = append(res.Items, m)
	}
	for _, par := range participants {
		p := &Participant{
			ID: par.Person.Id,
		}
		switch par.Person.RoleType {
		case api.PATIENT_ROLE:
			p.Name = fmt.Sprintf("%s %s", par.Person.Patient.FirstName, par.Person.Patient.LastName)
			if len(par.Person.Patient.FirstName) > 0 {
				p.Initials += par.Person.Patient.FirstName[:1]
			}
			if len(par.Person.Patient.LastName) > 0 {
				p.Initials += par.Person.Patient.LastName[:1]
			}
		case api.DOCTOR_ROLE, api.MA_ROLE:
			p.Name = par.Person.Doctor.LongDisplayName
			if len(par.Person.Doctor.FirstName) > 0 {
				p.Initials += par.Person.Doctor.FirstName[:1]
			}
			if len(par.Person.Doctor.LastName) > 0 {
				p.Initials += par.Person.Doctor.LastName[:1]
			}
			p.ThumbnailURL = par.Person.Doctor.SmallThumbnailURL
			p.Subtitle = par.Person.Doctor.ShortTitle
		}
		res.Participants = append(res.Participants, p)
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}

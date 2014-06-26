package messages

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Participant struct {
	ID           int64                `json:"participant_id,string"`
	Name         string               `json:"name"`
	Initials     string               `json:"initials"`
	Subtitle     string               `json:"subtitle,omitempty"`
	ThumbnailURL *app_url.SpruceAsset `json:"thumbnail_url,omitempty"`
}

type Message struct {
	Type        string        `json:"type"`
	Time        time.Time     `json:"date_time"`
	SenderID    int64         `json:"sender_participant_id,string"`
	Message     string        `json:"message"`
	Attachments []*Attachment `json:"attachments,omitempty"`
}

type ListResponse struct {
	Items        []*Message     `json:"items"`
	Participants []*Participant `json:"participants"`
}

type listHandler struct {
	dataAPI api.DataAPI
}

func NewListHandler(dataAPI api.DataAPI) http.Handler {
	return &listHandler{dataAPI: dataAPI}
}

func (h *listHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	caseID, err := strconv.ParseInt(r.FormValue("case_id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	cas, err := h.dataAPI.GetPatientCaseFromId(caseID)
	if err == api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusNotFound, "Case with the given ID does not exist")
		return
	}

	if _, _, err := validateAccess(h.dataAPI, r, cas); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	msgs, err := h.dataAPI.ListCaseMessages(caseID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	participants, err := h.dataAPI.CaseMessageParticipants(caseID, true)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &ListResponse{}
	for _, msg := range msgs {
		m := &Message{
			Type:     "conversation_item:message",
			Time:     msg.Time,
			SenderID: msg.PersonID,
			Message:  msg.Body,
		}

		for _, att := range msg.Attachments {
			a := &Attachment{
				Type: "attachment:" + att.ItemType,
				ID:   att.ItemID,
			}

			switch att.ItemType {
			case common.AttachmentTypePhoto:
				a.URL = apiservice.CreatePhotoUrl(att.ItemID, msg.ID, common.ClaimerTypeConversationMessage, r.Host)
			case common.AttachmentTypeTreatmentPlan:
				a.URL = app_url.ViewTreatmentPlanAction(att.ItemID).String()
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
		case api.DOCTOR_ROLE:
			p.Name = fmt.Sprintf("%s %s", par.Person.Doctor.FirstName, par.Person.Doctor.LastName)
			if len(par.Person.Doctor.FirstName) > 0 {
				p.Initials += par.Person.Doctor.FirstName[:1]
			}
			if len(par.Person.Doctor.LastName) > 0 {
				p.Initials += par.Person.Doctor.LastName[:1]
			}
			p.ThumbnailURL = par.Person.Doctor.SmallThumbnailUrl
			p.Subtitle = "Dermatologist" // TODO: update this once we have titles for doctors
		}
		res.Participants = append(res.Participants, p)
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}

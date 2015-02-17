package messages

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
)

type Participant struct {
	ID           int64  `json:"participant_id,string"`
	Name         string `json:"name"`
	Initials     string `json:"initials"`
	Subtitle     string `json:"subtitle,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
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
	dataAPI            api.DataAPI
	apiDomain          string
	mediaStore         *media.Store
	expirationDuration time.Duration
}

func NewListHandler(
	dataAPI api.DataAPI,
	apiDomain string,
	mediaStore *media.Store,
	expirationDuration time.Duration) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&listHandler{
				dataAPI:            dataAPI,
				apiDomain:          apiDomain,
				mediaStore:         mediaStore,
				expirationDuration: expirationDuration,
			}), []string{"GET"})
}

func (h *listHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	caseID, err := strconv.ParseInt(r.FormValue("case_id"), 10, 64)
	if err != nil {
		return false, apiservice.NewValidationError("bad case_id")
	}

	cas, err := h.dataAPI.GetPatientCaseFromID(caseID)
	if api.IsErrNotFound(err) {
		return false, apiservice.NewResourceNotFoundError("Case not found", r)
	} else if err != nil {
		return false, err
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

	msgs, err := h.dataAPI.ListCaseMessages(cas.ID.Int64(), ctxt.Role)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	participants, err := h.dataAPI.CaseMessageParticipants(cas.ID.Int64(), true)
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
				Type:  AttachmentTypePrefix + att.ItemType,
				ID:    att.ItemID,
				Title: att.Title,
			}
			switch att.ItemType {
			case common.AttachmentTypeFollowupVisit:
				a.URL = app_url.ContinueVisitAction(att.ItemID).String()
			case common.AttachmentTypeTreatmentPlan:
				a.URL = app_url.ViewTreatmentPlanAction(att.ItemID).String()
			case common.AttachmentTypeVisit:
				a.URL = app_url.ContinueVisitAction(att.ItemID).String()
			case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
				if ok, err := h.dataAPI.MediaHasClaim(att.ItemID, common.ClaimerTypeConversationMessage, msg.ID); err != nil {
					apiservice.WriteError(err, w, r)
				} else if !ok {
					// This should never happen but best to make sure
					golog.Errorf("Message %d attachment %d references media %d which it does not own", msg.ID, att.ID, att.ItemID)
					continue
				}

				a.MimeType = att.MimeType
				a.URL, err = h.mediaStore.SignedURL(att.ItemID, h.expirationDuration)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
			}

			m.Attachments = append(m.Attachments, a)
		}

		res.Items = append(res.Items, m)
	}
	for _, par := range participants {
		p := &Participant{
			ID: par.Person.ID,
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
			p.ThumbnailURL = app_url.ThumbnailURL(h.apiDomain, par.Person.RoleType, par.Person.Doctor.DoctorID.Int64())
			p.Subtitle = par.Person.Doctor.ShortTitle
		}
		res.Participants = append(res.Participants, p)
	}

	httputil.JSONResponse(w, http.StatusOK, res)
}

func AttachmentTitle(typ string) string {
	switch typ {
	case common.AttachmentTypeFollowupVisit:
		return "Follow-Up Visit"
	case common.AttachmentTypeTreatmentPlan:
		return "View Treatment Plan"
	case common.AttachmentTypeVisit:
		return "View Visit"
	}
	return ""
}

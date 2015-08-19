package messages

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
	"golang.org/x/net/context"
)

type Participant struct {
	ID           int64  `json:"participant_id,string"`
	Name         string `json:"name"`
	Initials     string `json:"initials"`
	Subtitle     string `json:"subtitle,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

type Message struct {
	ID           int64          `json:"message_id,string"`
	Type         string         `json:"type"`
	Time         time.Time      `json:"date_time"`
	Timestamp    int64          `json:"timestamp"`
	SenderID     int64          `json:"sender_participant_id,string"`
	Message      string         `json:"message"`
	Attachments  []*Attachment  `json:"attachments,omitempty"`
	StatusText   string         `json:"status_text,omitempty"`
	ReadReceipts []*ReadReceipt `json:"read_receipts,omitempty"`
}

type ReadReceipt struct {
	ParticipantID int64 `json:"participant_id,string"`
	Timestamp     int64 `json:"timestamp"`
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
	expirationDuration time.Duration) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(
				&listHandler{
					dataAPI:            dataAPI,
					apiDomain:          apiDomain,
					mediaStore:         mediaStore,
					expirationDuration: expirationDuration,
				})),
		httputil.Get)
}

func (h *listHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)

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
	requestCache[apiservice.CKPatientCase] = cas

	personID, _, err := validateAccess(h.dataAPI, r, apiservice.MustCtxAccount(ctx), cas)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPersonID] = personID

	return true, nil
}

func (h *listHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)
	requestCache := apiservice.MustCtxCache(ctx)
	cas := requestCache[apiservice.CKPatientCase].(*common.PatientCase)

	var lcmOpts api.ListCaseMessagesOption
	switch account.Role {
	case api.RoleDoctor:
		lcmOpts |= api.LCMOIncludePrivate
	case api.RoleCC:
		lcmOpts |= api.LCMOIncludePrivate | api.LCMOIncludeReadReceipts
	}

	var msgs []*common.CaseMessage
	var participants map[int64]*common.CaseMessageParticipant

	p := conc.NewParallel()

	// get case messages
	p.Go(func() error {
		var err error
		msgs, err = h.dataAPI.ListCaseMessages(cas.ID.Int64(), lcmOpts)
		return err
	})

	// get case message participants
	p.Go(func() error {
		var err error
		participants, err = h.dataAPI.CaseMessageParticipants(cas.ID.Int64(), true)
		return err
	})

	// get all visits associated with the case
	var visitMap map[int64]*common.PatientVisit
	p.Go(func() error {
		visits, err := h.dataAPI.GetVisitsForCase(cas.ID.Int64(), nil)
		if err != nil {
			return err
		}

		visitMap = make(map[int64]*common.PatientVisit, len(visits))
		for _, visit := range visits {
			visitMap[visit.ID.Int64()] = visit
		}
		return nil
	})

	if err := p.Wait(); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if account.Role == api.RoleCC {
		// Look up any people in the read receipts that we don't already have as a participant.
		var peopleIDs []int64
		for _, m := range msgs {
			for _, rr := range m.ReadReceipts {
				if _, ok := participants[rr.PersonID]; !ok {
					peopleIDs = append(peopleIDs, rr.PersonID)
				}
			}
		}
		if len(peopleIDs) != 0 {
			rrPeople, err := h.dataAPI.GetPeople(peopleIDs)
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}
			if participants == nil {
				participants = make(map[int64]*common.CaseMessageParticipant, len(rrPeople))
			}
			for _, p := range rrPeople {
				participants[p.ID] = &common.CaseMessageParticipant{
					CaseID: cas.ID.Int64(),
					Person: p,
				}
			}
		}
	}

	res := &ListResponse{
		Items: make([]*Message, 0, len(msgs)),
	}
	msgIDs := make([]int64, 0, len(msgs))
	for _, msg := range msgs {
		msgType := "conversation_item:message"
		if msg.IsPrivate {
			msgType = "conversation_item:private_message"
		}

		m := &Message{
			ID:          msg.ID,
			Type:        msgType,
			Time:        msg.Time,
			Timestamp:   msg.Time.Unix(),
			SenderID:    msg.PersonID,
			Message:     msg.Body,
			StatusText:  msg.EventText,
			Attachments: make([]*Attachment, 0, len(msg.Attachments)),
		}

		if len(msg.ReadReceipts) != 0 {
			m.ReadReceipts = make([]*ReadReceipt, len(msg.ReadReceipts))
			for i, rr := range msg.ReadReceipts {
				m.ReadReceipts[i] = &ReadReceipt{
					ParticipantID: rr.PersonID,
					Timestamp:     rr.Time.Unix(),
				}
			}
		}

		for _, att := range msg.Attachments {
			a := &Attachment{
				Type:  AttachmentTypePrefix + att.ItemType,
				ID:    att.ItemID,
				Title: att.Title,
			}
			switch att.ItemType {
			case common.AttachmentTypeResourceGuide:
				a.URL = app_url.ViewResourceGuideAction(att.ItemID).String()
			case common.AttachmentTypeFollowupVisit:
				isSubmitted := true
				pv, ok := visitMap[att.ItemID]
				if !ok {
					golog.Errorf("Visit not found for case %d. Treating visit as being submitted", cas.ID.Int64())
				} else {
					isSubmitted = common.PatientVisitSubmitted(pv.Status)
				}
				a.URL = app_url.ContinueVisitAction(att.ItemID, isSubmitted).String()
			case common.AttachmentTypeTreatmentPlan:
				a.URL = app_url.ViewTreatmentPlanAction(att.ItemID).String()
			case common.AttachmentTypeVisit:
				isSubmitted := true
				pv, ok := visitMap[att.ItemID]
				if !ok {
					golog.Errorf("Visit not found for case %d. Treating visit as being submitted", cas.ID.Int64())
				} else {
					isSubmitted = common.PatientVisitSubmitted(pv.Status)
				}
				a.URL = app_url.ContinueVisitAction(att.ItemID, isSubmitted).String()
			case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
				if ok, err := h.dataAPI.MediaHasClaim(att.ItemID, common.ClaimerTypeConversationMessage, msg.ID); err != nil {
					apiservice.WriteError(ctx, err, w, r)
					return
				} else if !ok {
					// This should never happen but best to make sure
					golog.Errorf("Message %d attachment %d references media %d which it does not own", msg.ID, att.ID, att.ItemID)
					continue
				}

				a.MimeType = att.MimeType
				var err error
				a.URL, err = h.mediaStore.SignedURL(att.ItemID, h.expirationDuration)
				if err != nil {
					apiservice.WriteError(ctx, err, w, r)
					return
				}
			default:
				golog.Errorf("Unknown attachment type %s for message %d", att.ItemType, msg.ID)
			}

			m.Attachments = append(m.Attachments, a)
		}

		res.Items = append(res.Items, m)
		msgIDs = append(msgIDs, m.ID)
	}
	for _, par := range participants {
		p := &Participant{
			ID: par.Person.ID,
		}
		switch par.Person.RoleType {
		case api.RolePatient:
			p.Name = fullName(par.Person.Patient.FirstName, par.Person.Patient.LastName)
			p.Initials = initials(par.Person.Patient.FirstName, par.Person.Patient.LastName)
		case api.RoleDoctor, api.RoleCC:
			p.Name = par.Person.Doctor.LongDisplayName
			p.Initials = initials(par.Person.Doctor.FirstName, par.Person.Doctor.LastName)
			p.ThumbnailURL = app_url.ThumbnailURL(h.apiDomain, par.Person.RoleType, par.Person.Doctor.ID.Int64())
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

func initials(firstName, lastName string) string {
	var ins string
	if firstName != "" {
		ins = firstName[:1]
	}
	if lastName != "" {
		ins += lastName[:1]
	}
	return strings.ToUpper(ins)
}

func fullName(firstName, lastName string) string {
	return strings.TrimSpace(firstName + " " + lastName)
}

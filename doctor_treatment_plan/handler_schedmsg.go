package doctor_treatment_plan

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/messages"
)

type scheduledMessageRequest interface {
	Validate() string
	TPID() int64
}

type scheduledMessageHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

type scheduledMessageIDRequest struct {
	TreatmentPlanID int64 `json:"treatment_plan_id,string" schema:"treatment_plan_id"`
	MessageID       int64 `json:"message_id,string" schema:"message_id"`
}

func (r *scheduledMessageIDRequest) Validate() string {
	if r.TreatmentPlanID <= 0 {
		return "treatment_plan_id is required"
	}
	return ""
}

func (r *scheduledMessageIDRequest) TPID() int64 {
	return r.TreatmentPlanID
}

var scheduledMessageReqTypes = map[string]scheduledMessageRequest{
	"GET":    &scheduledMessageIDRequest{},
	"DELETE": &scheduledMessageIDRequest{},
	"POST":   &ScheduledMessageRequest{},
	"PUT":    &ScheduledMessageRequest{},
}

type ScheduledMessage struct {
	ID            int64                  `json:"id,string"`
	Title         string                 `json:"title"`
	ScheduledDays int                    `json:"scheduled_days"`
	ScheduledFor  time.Time              `json:"scheduled_for"`
	Message       string                 `json:"message"`
	Attachments   []*messages.Attachment `json:"attachments"`
}

type ScheduledMessageRequest struct {
	TreatmentPlanID int64             `json:"treatment_plan_id,string"`
	Message         *ScheduledMessage `json:"scheduled_message"`
}

func (r *ScheduledMessageRequest) TPID() int64 {
	return r.TreatmentPlanID
}

func (r *ScheduledMessageRequest) Validate() string {
	if r.TreatmentPlanID <= 0 {
		return "treatment_plan_id is required"
	}
	sm := r.Message
	if sm == nil {
		return "scheduled message is required"
	}
	if sm.ScheduledDays <= 0 {
		return "scheduled_days is required"
	}
	if sm.Message == "" {
		return "message is required"
	}
	for _, a := range sm.Attachments {
		// Strip "attachment:" prefix on type if necessary
		if idx := strings.IndexByte(a.Type, ':'); idx >= 0 {
			a.Type = a.Type[idx+1:]
		}

		switch a.Type {
		case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
			if a.ID <= 0 {
				return "id is required for attachment types photo and audio"
			}
		case common.AttachmentTypeFollowupVisit:
		default:
			// Only allow the above explicitely listed attachments. The other ones
			// (e.g. treatment plan, continue visit) don't make sense for scheduled
			// messages.
			return "attachment type not allowed"
		}
	}
	return ""
}

type ScheduledMessageListResponse struct {
	Messages []*ScheduledMessage `json:"scheduled_messages"`
}

type ScheduledMessageIDResponse struct {
	MessageID int64 `json:"message_id,string"`
}

func NewScheduledMessageHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&scheduledMessageHandler{
					dataAPI:    dataAPI,
					dispatcher: dispatcher,
				}),
			[]string{api.DOCTOR_ROLE, api.MA_ROLE}),
		[]string{"GET", "POST", "PUT", "DELETE"})
}

func (h *scheduledMessageHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctx := apiservice.GetContext(r)

	// Decode and validate request body or parameters (depending on method)

	req := reflect.New(reflect.TypeOf(scheduledMessageReqTypes[r.Method]).Elem()).Interface().(scheduledMessageRequest)
	if err := apiservice.DecodeRequestData(req, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	if e := req.Validate(); e != "" {
		return false, apiservice.NewValidationError(e, r)
	}
	ctx.RequestCache[apiservice.RequestData] = req

	// Validate authorization

	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(ctx.AccountID)
	if err != nil {
		return false, err
	}
	ctx.RequestCache[apiservice.DoctorID] = doctorID

	tp, err := h.dataAPI.GetAbridgedTreatmentPlan(req.TPID(), doctorID)
	if err != nil {
		return false, err
	}
	ctx.RequestCache[apiservice.TreatmentPlan] = tp

	// If trying to update, make sure doctor owns the treatment plan
	if r.Method != "GET" {
		if tp.DoctorID.Int64() != doctorID {
			return false, apiservice.NewAccessForbiddenError()
		}
		// Only allow editing draft treatment plans
		if !tp.InDraftMode() {
			return false, apiservice.NewValidationError("treatment plan must be a draft", r)
		}
	}

	// Make sure doctor is allowed to work on the case
	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctx.Role, doctorID, tp.PatientID, tp.PatientCaseID.Int64(), h.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (h *scheduledMessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getMessages(w, r)
	case "POST":
		h.createMessage(w, r)
	case "PUT":
		h.updateMessage(w, r)
	case "DELETE":
		h.deleteMessage(w, r)
	}
}

func (h *scheduledMessageHandler) getMessages(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)
	req := ctx.RequestCache[apiservice.RequestData].(*scheduledMessageIDRequest)
	tp := ctx.RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

	msgs, err := h.dataAPI.ListTreatmentPlanScheduledMessages(req.TreatmentPlanID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &ScheduledMessageListResponse{
		Messages: make([]*ScheduledMessage, len(msgs)),
	}

	now := time.Now()

	for i, m := range msgs {
		msg := &ScheduledMessage{
			ID:            m.ID,
			ScheduledDays: m.ScheduledDays,
			Message:       m.Message,
			Attachments:   make([]*messages.Attachment, len(m.Attachments)),
		}
		res.Messages[i] = msg

		if tp.SentDate != nil {
			msg.ScheduledFor = tp.SentDate.Add(24 * time.Hour * time.Duration(m.ScheduledDays))
		} else {
			msg.ScheduledFor = now.Add(24 * time.Hour * time.Duration(m.ScheduledDays))
		}

		for j, a := range m.Attachments {
			att := &messages.Attachment{
				ID:       a.ItemID,
				Type:     messages.AttachmentTypePrefix + a.ItemType,
				Title:    a.Title,
				MimeType: a.MimeType,
			}
			msg.Attachments[j] = att
		}

		msg.Title = titleForScheduledMessage(msg)
	}

	apiservice.WriteJSON(w, res)
}

func (h *scheduledMessageHandler) createMessage(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)
	req := ctx.RequestCache[apiservice.RequestData].(*ScheduledMessageRequest)

	msg, err := h.transformMessage(r, ctx, req.Message, req.TreatmentPlanID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// For an existing messages that matches exactly (makes POST idempotent)
	msgs, err := h.dataAPI.ListTreatmentPlanScheduledMessages(req.TreatmentPlanID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	for _, m := range msgs {
		if m.Equal(msg) {
			apiservice.WriteJSON(w, &ScheduledMessageIDResponse{MessageID: m.ID})
			return
		}
	}

	msgID, err := h.dataAPI.CreateTreatmentPlanScheduledMessage(msg)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanScheduledMessagesUpdatedEvent{
		DoctorID:        ctx.RequestCache[apiservice.DoctorID].(int64),
		TreatmentPlanID: req.TreatmentPlanID,
	})

	apiservice.WriteJSON(w, &ScheduledMessageIDResponse{MessageID: msgID})
}

func (h *scheduledMessageHandler) updateMessage(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)
	req := ctx.RequestCache[apiservice.RequestData].(*ScheduledMessageRequest)
	if req.Message.ID <= 0 {
		apiservice.WriteBadRequestError(errors.New("id is required"), w, r)
		return
	}
	msg, err := h.transformMessage(r, ctx, req.Message, req.TreatmentPlanID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	if err := h.dataAPI.ReplaceTreatmentPlanScheduledMessage(req.Message.ID, msg); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanScheduledMessagesUpdatedEvent{
		DoctorID:        ctx.RequestCache[apiservice.DoctorID].(int64),
		TreatmentPlanID: req.TreatmentPlanID,
	})

	apiservice.WriteJSON(w, &ScheduledMessageIDResponse{MessageID: msg.ID})
}

func (h *scheduledMessageHandler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)
	req := ctx.RequestCache[apiservice.RequestData].(*scheduledMessageIDRequest)
	if req.MessageID == 0 {
		apiservice.WriteBadRequestError(errors.New("message_id is required"), w, r)
		return
	}

	if err := h.dataAPI.DeleteTreatmentPlanScheduledMessage(req.TreatmentPlanID, req.MessageID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanScheduledMessagesUpdatedEvent{
		DoctorID:        ctx.RequestCache[apiservice.DoctorID].(int64),
		TreatmentPlanID: req.TreatmentPlanID,
	})

	apiservice.WriteJSONSuccess(w)
}

func (h *scheduledMessageHandler) transformMessage(r *http.Request, ctx *apiservice.Context, msg *ScheduledMessage, tpID int64) (*common.TreatmentPlanScheduledMessage, error) {
	m := &common.TreatmentPlanScheduledMessage{
		TreatmentPlanID: tpID,
		ScheduledDays:   msg.ScheduledDays,
		Attachments:     make([]*common.CaseMessageAttachment, len(msg.Attachments)),
		Message:         msg.Message,
	}
	var personID int64
	for i, a := range msg.Attachments {
		switch a.Type {
		case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
			// Delayed querying of person ID (only needed when checking media)
			if personID == 0 {
				var err error
				personID, err = h.dataAPI.GetPersonIDByRole(ctx.Role, ctx.RequestCache[apiservice.DoctorID].(int64))
				if err != nil {
					return nil, err
				}
			}

			// Make sure media is uploaded by the same person and is unclaimed
			media, err := h.dataAPI.GetMedia(a.ID)
			if err != nil {
				return nil, err
			}
			if media.UploaderID != personID {
				return nil, apiservice.NewValidationError("invalid attached media", r)
			}
		case common.AttachmentTypeFollowupVisit:
		default:
			return nil, apiservice.NewValidationError("attachment type "+a.Type+" not allowed in scheduled message", r)
		}

		title := a.Title
		if title == "" {
			title = messages.AttachmentTitle(a.Type)
		}

		m.Attachments[i] = &common.CaseMessageAttachment{
			Title:    title,
			ItemID:   a.ID,
			ItemType: a.Type,
			MimeType: a.MimeType,
		}
	}
	return m, nil
}

func titleForScheduledMessage(m *ScheduledMessage) string {
	isFollowUp := false
	for _, a := range m.Attachments {
		if a.Type == messages.AttachmentTypePrefix+common.AttachmentTypeFollowupVisit {
			isFollowUp = true
			break
		}
	}

	var humanTime string
	days := m.ScheduledFor.Sub(time.Now()) / (time.Hour * 24)
	if days <= 1 {
		humanTime = "1 day"
	} else if days < 7 {
		humanTime = fmt.Sprintf("%d days", days)
	} else if days < 14 {
		humanTime = "1 week"
	} else {
		humanTime = fmt.Sprintf("%d weeks", days/7)
	}

	if isFollowUp {
		return "Message & Follow-Up Visit in " + humanTime
	}
	return "Message in " + humanTime
}

package doctor_treatment_plan

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/responses"
)

const scheduledMessageMediaExpirationDuration = time.Hour * 24 * 7

type scheduledMessageRequest interface {
	Validate() string
	TPID() int64
}

type scheduledMessageHandler struct {
	dataAPI    api.DataAPI
	mediaStore storage.Store
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

type ScheduledMessageRequest struct {
	TreatmentPlanID int64                       `json:"treatment_plan_id,string"`
	Message         *responses.ScheduledMessage `json:"scheduled_message"`
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
	Messages []*responses.ScheduledMessage `json:"scheduled_messages"`
}

type ScheduledMessageIDResponse struct {
	MessageID int64 `json:"message_id,string"`
}

func NewScheduledMessageHandler(dataAPI api.DataAPI, mediaStore storage.Store, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&scheduledMessageHandler{
					dataAPI:    dataAPI,
					mediaStore: mediaStore,
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
		return false, apiservice.NewValidationError(err.Error())
	}
	if e := req.Validate(); e != "" {
		return false, apiservice.NewValidationError(e)
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
			return false, apiservice.NewValidationError("treatment plan must be a draft")
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
		Messages: make([]*responses.ScheduledMessage, len(msgs)),
	}

	var sent time.Time
	if tp.SentDate != nil {
		sent = *tp.SentDate
	} else {
		sent = time.Now()
	}

	for i, m := range msgs {
		res.Messages[i], err = responses.TransformScheduledMessageToResponse(
			h.dataAPI,
			h.mediaStore,
			m,
			sent,
			scheduledMessageMediaExpirationDuration)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSON(w, res)
}

func (h *scheduledMessageHandler) createMessage(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)
	req := ctx.RequestCache[apiservice.RequestData].(*ScheduledMessageRequest)

	doctorID := ctx.RequestCache[apiservice.DoctorID].(int64)
	msg, err := responses.TransformScheduledMessageFromResponse(
		h.dataAPI,
		req.Message,
		req.TreatmentPlanID,
		doctorID,
		ctx.Role)
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

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		SectionUpdated:  ScheduledMessagesSection,
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
	doctorID := ctx.RequestCache[apiservice.DoctorID].(int64)
	msg, err := responses.TransformScheduledMessageFromResponse(
		h.dataAPI,
		req.Message,
		req.TreatmentPlanID,
		doctorID,
		ctx.Role)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	if err := h.dataAPI.ReplaceTreatmentPlanScheduledMessage(req.Message.ID, msg); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		SectionUpdated:  ScheduledMessagesSection,
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

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		SectionUpdated:  ScheduledMessagesSection,
		DoctorID:        ctx.RequestCache[apiservice.DoctorID].(int64),
		TreatmentPlanID: req.TreatmentPlanID,
	})

	apiservice.WriteJSONSuccess(w)
}

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
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/responses"
	"golang.org/x/net/context"
)

const scheduledMessageMediaExpirationDuration = time.Hour * 24 * 7

type scheduledMessageRequest interface {
	Validate() string
	TPID() int64
}

type scheduledMessageHandler struct {
	dataAPI    api.DataAPI
	mediaStore *media.Store
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

func NewScheduledMessageHandler(dataAPI api.DataAPI, mediaStore *media.Store, dispatcher *dispatch.Dispatcher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(
					&scheduledMessageHandler{
						dataAPI:    dataAPI,
						mediaStore: mediaStore,
						dispatcher: dispatcher,
					})),
			api.RoleDoctor, api.RoleCC),
		httputil.Get, httputil.Post, httputil.Put, httputil.Delete)
}

func (h *scheduledMessageHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	// Decode and validate request body or parameters (depending on method)

	req := reflect.New(reflect.TypeOf(scheduledMessageReqTypes[r.Method]).Elem()).Interface().(scheduledMessageRequest)
	if err := apiservice.DecodeRequestData(req, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	if e := req.Validate(); e != "" {
		return false, apiservice.NewValidationError(e)
	}
	requestCache[apiservice.CKRequestData] = req

	// Validate authorization

	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctorID] = doctorID

	tp, err := h.dataAPI.GetAbridgedTreatmentPlan(req.TPID(), doctorID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKTreatmentPlan] = tp

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
	if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID, tp.PatientID, tp.PatientCaseID.Int64(), h.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (h *scheduledMessageHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getMessages(ctx, w, r)
	case "POST":
		h.createMessage(ctx, w, r)
	case "PUT":
		h.updateMessage(ctx, w, r)
	case "DELETE":
		h.deleteMessage(ctx, w, r)
	}
}

func (h *scheduledMessageHandler) getMessages(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	req := requestCache[apiservice.CKRequestData].(*scheduledMessageIDRequest)
	tp := requestCache[apiservice.CKTreatmentPlan].(*common.TreatmentPlan)

	msgs, err := h.dataAPI.ListTreatmentPlanScheduledMessages(req.TreatmentPlanID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	httputil.JSONResponse(w, http.StatusOK, res)
}

func (h *scheduledMessageHandler) createMessage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	req := requestCache[apiservice.CKRequestData].(*ScheduledMessageRequest)
	account := apiservice.MustCtxAccount(ctx)

	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	msg, err := responses.TransformScheduledMessageFromResponse(
		h.dataAPI,
		req.Message,
		req.TreatmentPlanID,
		doctorID,
		account.Role)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// For an existing messages that matches exactly (makes POST idempotent)
	msgs, err := h.dataAPI.ListTreatmentPlanScheduledMessages(req.TreatmentPlanID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	for _, m := range msgs {
		if m.Equal(msg) {
			httputil.JSONResponse(w, http.StatusOK, &ScheduledMessageIDResponse{MessageID: m.ID})
			return
		}
	}

	msgID, err := h.dataAPI.CreateTreatmentPlanScheduledMessage(msg)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		SectionUpdated:  ScheduledMessagesSection,
		DoctorID:        requestCache[apiservice.CKDoctorID].(int64),
		TreatmentPlanID: req.TreatmentPlanID,
	})
	httputil.JSONResponse(w, http.StatusOK, &ScheduledMessageIDResponse{MessageID: msgID})
}

func (h *scheduledMessageHandler) updateMessage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)
	requestCache := apiservice.MustCtxCache(ctx)
	req := requestCache[apiservice.CKRequestData].(*ScheduledMessageRequest)
	if req.Message.ID <= 0 {
		apiservice.WriteBadRequestError(ctx, errors.New("id is required"), w, r)
		return
	}
	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	msg, err := responses.TransformScheduledMessageFromResponse(
		h.dataAPI,
		req.Message,
		req.TreatmentPlanID,
		doctorID,
		account.Role)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	if err := h.dataAPI.ReplaceTreatmentPlanScheduledMessage(req.Message.ID, msg); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		SectionUpdated:  ScheduledMessagesSection,
		DoctorID:        requestCache[apiservice.CKDoctorID].(int64),
		TreatmentPlanID: req.TreatmentPlanID,
	})

	httputil.JSONResponse(w, http.StatusOK, &ScheduledMessageIDResponse{MessageID: msg.ID})
}

func (h *scheduledMessageHandler) deleteMessage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	req := requestCache[apiservice.CKRequestData].(*scheduledMessageIDRequest)
	if req.MessageID == 0 {
		apiservice.WriteBadRequestError(ctx, errors.New("message_id is required"), w, r)
		return
	}

	if err := h.dataAPI.DeleteTreatmentPlanScheduledMessage(req.TreatmentPlanID, req.MessageID); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		SectionUpdated:  ScheduledMessagesSection,
		DoctorID:        requestCache[apiservice.CKDoctorID].(int64),
		TreatmentPlanID: req.TreatmentPlanID,
	})

	apiservice.WriteJSONSuccess(w)
}

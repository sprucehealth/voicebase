package messages

import (
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
)

type handler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

type PostMessageRequest struct {
	CaseID      int64         `json:"case_id,string"`
	Message     string        `json:"message"`
	Attachments []*Attachment `json:"attachments,omitempty"`
}

func (r *PostMessageRequest) Validate() error {
	if r.CaseID <= 0 {
		return errors.New("case_id missing or invalid")
	}
	if r.Message == "" {
		return errors.New("message must not be blank")
	}
	return nil
}

type PostMessageResponse struct {
	MessageID int64 `json:"message_id,string"`
}

type Attachment struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	MimeType string `json:"mimetype,omitempty"`
	ID       int64  `json:"id,string"`
	URL      string `json:"url,omitempty"`
}

func NewHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(
				&handler{
					dataAPI:    dataAPI,
					dispatcher: dispatcher})),
		httputil.Post)
}

func (h *handler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)

	var req PostMessageRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = &req

	if err := req.Validate(); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}

	cas, err := h.dataAPI.GetPatientCaseFromID(req.CaseID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatientCase] = cas

	personID, doctorID, err := validateAccess(h.dataAPI, r, apiservice.MustCtxAccount(ctx), cas)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPersonID] = personID
	requestCache[apiservice.CKDoctorID] = doctorID

	return true, nil
}

func (h *handler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	req := requestCache[apiservice.CKRequestData].(*PostMessageRequest)
	personID := requestCache[apiservice.CKPersonID].(int64)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	cas := requestCache[apiservice.CKPatientCase].(*common.PatientCase)

	people, err := h.dataAPI.GetPeople([]int64{personID})
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	person := people[personID]

	msg := &common.CaseMessage{
		CaseID:   req.CaseID,
		PersonID: personID,
		Body:     req.Message,
	}

	account := apiservice.MustCtxAccount(ctx)
	if err := CreateMessageAndAttachments(msg, req.Attachments, personID, doctorID, account.Role, h.dataAPI); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	h.dispatcher.Publish(&PostEvent{
		Message: msg,
		Case:    cas,
		Person:  person,
	})

	res := &PostMessageResponse{
		MessageID: msg.ID,
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

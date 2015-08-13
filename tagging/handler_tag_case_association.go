package tagging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/tagging/model"
	"github.com/sprucehealth/backend/tagging/query"
	"github.com/sprucehealth/backend/tagging/response"
)

type tagCaseAssociationHandler struct {
	taggingClient Client
}

// TagCaseAssociationGETRequest represents the data expected to be associated with a sucessful GET request
type TagCaseAssociationGETRequest struct {
	Query       string `schema:"query"`
	Start       int64  `schema:"start,required"`
	End         int64  `schema:"end"`
	PastTrigger bool   `schema:"past_trigger"`
}

// TagCaseAssociationGETResponse represents the data expected to be returned from a sucessful GET request
type TagCaseAssociationGETResponse struct {
	Associations []*response.TagAssociation `json:"associations"`
}

// TagCaseAssociationPOSTRequest represents the data expected to be associated with a sucessful POST request
type TagCaseAssociationPOSTRequest struct {
	Text        string `json:"text"`
	Common      bool   `json:"common"`
	CaseID      *int64 `json:"case_id,string"`
	TriggerTime *int64 `json:"trigger_time"`
	Hidden      bool   `json:"hidden"`
}

// TagCaseAssociationPOSTResponse represents the data expected to be returned from a sucessful POST request
type TagCaseAssociationPOSTResponse struct {
	TagID int64 `json:"tag_id,string"`
}

// TagCaseAssociationDELETERequest represents the data expected to be associated with a sucessful DELETE request
type TagCaseAssociationDELETERequest struct {
	Text   string `schema:"text,required"`
	CaseID int64  `schema:"case_id,required"`
}

// NewTagCaseAssociationHandler returns an initialized instance of tagCaseAssociationHandler
func NewTagCaseAssociationHandler(taggingClient Client) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&tagCaseAssociationHandler{taggingClient: taggingClient}),
			api.RoleCC),
		httputil.Get, httputil.Post, httputil.Delete)
}

func (h *tagCaseAssociationHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		req, err := h.parsePOSTRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.servePOST(ctx, w, r, req)
	case "GET":
		req, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.serveGET(ctx, w, r, req)
	case "DELETE":
		req, err := h.parseDELETERequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.serveDELETE(ctx, w, r, req)
	}
}

func (h *tagCaseAssociationHandler) parseGETRequest(ctx context.Context, r *http.Request) (*TagCaseAssociationGETRequest, error) {
	rd := &TagCaseAssociationGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagCaseAssociationHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, req *TagCaseAssociationGETRequest) {
	if len(strings.TrimSpace(req.Query)) == 0 && !req.PastTrigger {
		httputil.JSONResponse(w, http.StatusOK, &TagCaseAssociationGETResponse{
			Associations: []*response.TagAssociation{},
		})
		return
	}

	ops := TONone
	if req.PastTrigger {
		ops = TOPastTrigger
	}
	memberships, err := h.taggingClient.TagMembershipQuery(req.Query, ops)
	if query.IsErrBadExpression(err) {
		apiservice.WriteBadRequestError(ctx, err, w, r)
		return
	}
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	associations, err := h.taggingClient.CaseAssociations(memberships, req.Start, req.End)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &TagCaseAssociationGETResponse{
		Associations: associations,
	})
}

func (h *tagCaseAssociationHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*TagCaseAssociationPOSTRequest, error) {
	rd := &TagCaseAssociationPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.CaseID == nil {
		return nil, fmt.Errorf("At least 1 associated entitied required for tag creation")
	}
	return rd, nil
}

func (h *tagCaseAssociationHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, req *TagCaseAssociationPOSTRequest) {
	membership := &model.TagMembership{
		CaseID: req.CaseID,
		Hidden: req.Hidden,
	}
	if req.TriggerTime != nil {
		t := time.Unix(*req.TriggerTime, 0)
		membership.TriggerTime = &t
	}

	tagID, err := h.taggingClient.InsertTagAssociation(&model.Tag{
		Text:   req.Text,
		Common: req.Common,
	}, membership)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &TagCaseAssociationPOSTResponse{
		TagID: tagID,
	})
}

func (h *tagCaseAssociationHandler) parseDELETERequest(ctx context.Context, r *http.Request) (*TagCaseAssociationDELETERequest, error) {
	rd := &TagCaseAssociationDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagCaseAssociationHandler) serveDELETE(ctx context.Context, w http.ResponseWriter, r *http.Request, req *TagCaseAssociationDELETERequest) {
	if err := h.taggingClient.DeleteTagCaseAssociation(req.Text, req.CaseID); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

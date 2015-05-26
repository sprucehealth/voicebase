package tagging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
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

type TagCaseAssociationGETRequest struct {
	Query       string `schema:"query"`
	Start       int64  `schema:"start,required"`
	End         int64  `schema:"end"`
	PastTrigger bool   `schema:"past_trigger"`
}

type TagCaseAssociationGETResponse struct {
	Associations []*response.TagAssociation `json:"associations"`
}

type TagCaseAssociationPOSTRequest struct {
	Text        string `json:"text"`
	Common      bool   `json:"common"`
	CaseID      *int64 `json:"case_id,string"`
	TriggerTime *int64 `json:"trigger_time"`
	Hidden      bool   `json:"hidden"`
}

type TagCaseAssociationPOSTResponse struct {
	TagID int64 `json:"tag_id,string"`
}

type TagCaseAssociationDELETERequest struct {
	Text   string `schema:"text,required"`
	CaseID int64  `schema:"case_id,required"`
}

func NewTagCaseAssociationHandler(taggingClient Client) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&tagCaseAssociationHandler{taggingClient: taggingClient}),
		httputil.Get, httputil.Post, httputil.Delete)
}

func (p *tagCaseAssociationHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.RoleCC {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (h *tagCaseAssociationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		req, err := h.parsePOSTRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.servePOST(w, r, req)
	case "GET":
		req, err := h.parseGETRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.serveGET(w, r, req)
	case "DELETE":
		req, err := h.parseDELETERequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.serveDELETE(w, r, req)
	}
}

func (h *tagCaseAssociationHandler) parseGETRequest(r *http.Request) (*TagCaseAssociationGETRequest, error) {
	rd := &TagCaseAssociationGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagCaseAssociationHandler) serveGET(w http.ResponseWriter, r *http.Request, req *TagCaseAssociationGETRequest) {
	if len(strings.TrimSpace(req.Query)) == 0 && !req.PastTrigger {
		httputil.JSONResponse(w, http.StatusOK, &TagCaseAssociationGETResponse{
			Associations: []*response.TagAssociation{},
		})
		return
	}

	memberships, err := h.taggingClient.TagMembershipQuery(req.Query, req.PastTrigger)
	if query.IsErrBadExpression(err) {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	associations, err := h.taggingClient.CaseAssociations(memberships, req.Start, req.End)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &TagCaseAssociationGETResponse{
		Associations: associations,
	})
}

func (h *tagCaseAssociationHandler) parsePOSTRequest(r *http.Request) (*TagCaseAssociationPOSTRequest, error) {
	rd := &TagCaseAssociationPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.CaseID == nil {
		return nil, fmt.Errorf("At least 1 associated entitied required for tag creation")
	}
	return rd, nil
}

func (h *tagCaseAssociationHandler) servePOST(w http.ResponseWriter, r *http.Request, req *TagCaseAssociationPOSTRequest) {
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
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &TagCaseAssociationPOSTResponse{
		TagID: tagID,
	})
}

func (h *tagCaseAssociationHandler) parseDELETERequest(r *http.Request) (*TagCaseAssociationDELETERequest, error) {
	rd := &TagCaseAssociationDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagCaseAssociationHandler) serveDELETE(w http.ResponseWriter, r *http.Request, req *TagCaseAssociationDELETERequest) {
	if err := h.taggingClient.DeleteTagCaseAssociation(req.Text, req.CaseID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

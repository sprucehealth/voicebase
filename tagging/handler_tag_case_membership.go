package tagging

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/tagging/model"
	"github.com/sprucehealth/backend/tagging/response"

	"github.com/sprucehealth/backend/libs/httputil"
)

type tagCaseMembershipHandler struct {
	taggingClient Client
}

type TagCaseMembershipGETRequest struct {
	CaseID int64 `schema:"case_id,required"`
}

type TagCaseMembershipGETResponse struct {
	TagMemberships map[string]*response.TagMembership `json:"tag_memberships"`
}

type TagCaseMembershipDELETERequest struct {
	CaseID int64 `schema:"case_id,required"`
	TagID  int64 `schema:"tag_id,required"`
}

type TagCaseMembershipPUTRequest struct {
	CaseID      int64  `json:"case_id,string"`
	TagID       int64  `json:"tag_id,string"`
	TriggerTime *int64 `json:"trigger_time"`
}

func NewTagCaseMembershipHandler(taggingClient Client) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&tagCaseMembershipHandler{taggingClient: taggingClient}),
		httputil.Get, httputil.Delete, httputil.Put)
}

func (p *tagCaseMembershipHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.RoleCC {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (h *tagCaseMembershipHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
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
	case "PUT":
		req, err := h.parsePUTRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.servePUT(w, r, req)
	}
}

func (h *tagCaseMembershipHandler) parseGETRequest(r *http.Request) (*TagCaseMembershipGETRequest, error) {
	rd := &TagCaseMembershipGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagCaseMembershipHandler) serveGET(w http.ResponseWriter, r *http.Request, req *TagCaseMembershipGETRequest) {
	tagMemberships, err := h.taggingClient.CaseTagMemberships(req.CaseID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	tagMembershipResps := make(map[string]*response.TagMembership, len(tagMemberships))
	for tagText, m := range tagMemberships {
		tagMembershipResps[tagText] = response.TransformTagMembership(m)
	}

	httputil.JSONResponse(w, http.StatusOK, &TagCaseMembershipGETResponse{
		TagMemberships: tagMembershipResps,
	})
}

func (h *tagCaseMembershipHandler) parseDELETERequest(r *http.Request) (*TagCaseMembershipDELETERequest, error) {
	rd := &TagCaseMembershipDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagCaseMembershipHandler) serveDELETE(w http.ResponseWriter, r *http.Request, req *TagCaseMembershipDELETERequest) {
	if err := h.taggingClient.DeleteTagCaseMembership(req.TagID, req.CaseID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

func (h *tagCaseMembershipHandler) parsePUTRequest(r *http.Request) (*TagCaseMembershipPUTRequest, error) {
	rd := &TagCaseMembershipPUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.CaseID == 0 || rd.TagID == 0 {
		return nil, errors.New("case_id, tag_id required")
	}
	return rd, nil
}

func (h *tagCaseMembershipHandler) servePUT(w http.ResponseWriter, r *http.Request, req *TagCaseMembershipPUTRequest) {
	memUpdate := &model.TagMembershipUpdate{
		CaseID: &req.CaseID,
		TagID:  req.TagID,
	}
	if req.TriggerTime != nil {
		t := time.Unix(*req.TriggerTime, 0)
		memUpdate.TriggerTime = &t
	}
	if err := h.taggingClient.UpdateTagCaseMembership(memUpdate); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/tagging/model"
	"github.com/sprucehealth/backend/tagging/response"

	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type tagAssociationHandler struct {
	taggingClient tagging.Client
}

type tagAssociationGETRequest struct {
	Query string `schema:"query,required"`
}

type tagAssociationGETResponse struct {
	Associations []*response.TagAssociation `json:"associations"`
}

type tagAssociationPOSTRequest struct {
	Text        string `json:"text"`
	CaseID      *int64 `json:"case_id,string"`
	TriggerTime *int64 `json:"trigger_time,string"`
	Hidden      bool   `json:"hidden"`
}

type tagAssociationPOSTResponse struct {
	ID int64 `json:"id,string"`
}

type tagAssociationDELETERequest struct {
	Text   string `schema:"text,required"`
	CaseID int64  `schema:"case_id,required"`
}

func NewTagAssociationHandler(taggingClient tagging.Client) http.Handler {
	return httputil.SupportedMethods(&tagAssociationHandler{taggingClient: taggingClient}, []string{"GET", "POST", "DELETE"})
}

func (h *tagAssociationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		req, err := h.parsePOSTRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePOST(w, r, req)
	case "GET":
		req, err := h.parseGETRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveGET(w, r, req)
	case "DELETE":
		req, err := h.parseDELETERequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveDELETE(w, r, req)
	}
}

func (h *tagAssociationHandler) parseGETRequest(r *http.Request) (*tagAssociationGETRequest, error) {
	rd := &tagAssociationGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagAssociationHandler) serveGET(w http.ResponseWriter, r *http.Request, req *tagAssociationGETRequest) {
	if len(strings.TrimSpace(req.Query)) == 0 {
		httputil.JSONResponse(w, http.StatusOK, &tagAssociationGETResponse{
			Associations: []*response.TagAssociation{},
		})
		return
	}

	memberships, err := h.taggingClient.TagMembershipQuery(req.Query)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	associations, err := h.taggingClient.CaseAssociations(memberships)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &tagAssociationGETResponse{
		Associations: associations,
	})
}

func (h *tagAssociationHandler) parsePOSTRequest(r *http.Request) (*tagAssociationPOSTRequest, error) {
	rd := &tagAssociationPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.CaseID == nil {
		return nil, fmt.Errorf("At least 1 associated entitied required for tag creation")
	}
	return rd, nil
}

func (h *tagAssociationHandler) servePOST(w http.ResponseWriter, r *http.Request, req *tagAssociationPOSTRequest) {
	membership := &model.TagMembership{
		CaseID: req.CaseID,
		Hidden: req.Hidden,
	}
	if req.TriggerTime != nil {
		t := time.Unix(*req.TriggerTime, 0)
		membership.TriggerTime = &t
	}
	id, err := h.taggingClient.InsertTagAssociation(req.Text, membership)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &tagAssociationPOSTResponse{
		ID: id,
	})
}

func (h *tagAssociationHandler) parseDELETERequest(r *http.Request) (*tagAssociationDELETERequest, error) {
	rd := &tagAssociationDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagAssociationHandler) serveDELETE(w http.ResponseWriter, r *http.Request, req *tagAssociationDELETERequest) {
	if err := h.taggingClient.DeleteTagCaseAssociation(req.Text, req.CaseID); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, true)
}

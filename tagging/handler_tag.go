package tagging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/tagging/response"

	"github.com/sprucehealth/backend/libs/httputil"
)

type tagHandler struct {
	taggingClient Client
}

type tagGETRequest struct {
	Text []string `schema:"text,required"`
}

type tagGETResponse struct {
	Tags []*response.Tag `json:"tags"`
}

type tagDELETERequest struct {
	ID int64 `json:"id,string"`
}

func NewTagHandler(taggingClient Client) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&tagHandler{taggingClient: taggingClient}),
		httputil.Get, httputil.Delete)
}

func (p *tagHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.RoleMA {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (h *tagHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		req, err := h.parseDELETERequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.serveDELETE(w, r, req)
	case "GET":
		req, err := h.parseGETRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.serveGET(w, r, req)
	}
}

func (h *tagHandler) parseGETRequest(r *http.Request) (*tagGETRequest, error) {
	rd := &tagGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagHandler) serveGET(w http.ResponseWriter, r *http.Request, req *tagGETRequest) {
	text := make([]string, 0, len(req.Text))
	for _, v := range req.Text {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			text = append(text, trimmed)
		}
	}
	if len(text) == 0 {
		httputil.JSONResponse(w, http.StatusOK, &tagGETResponse{
			Tags: []*response.Tag{},
		})
		return
	}

	tags, err := h.taggingClient.Tags(text)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &tagGETResponse{
		Tags: tags,
	})
}

func (h *tagHandler) parseDELETERequest(r *http.Request) (*tagDELETERequest, error) {
	rd := &tagDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagHandler) serveDELETE(w http.ResponseWriter, r *http.Request, req *tagDELETERequest) {
	_, err := h.taggingClient.DeleteTag(req.ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

package tagging

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/tagging/response"
	"github.com/sprucehealth/schema"
)

type tagHandler struct {
	taggingClient Client
}

type TagGETRequest struct {
	Text   []string `schema:"text"`
	Common bool     `schema:"common"`
}

type TagGETResponse struct {
	Tags []*response.Tag `json:"tags"`
}

type TagDELETERequest struct {
	ID int64 `schema:"id,required"`
}

func NewTagHandler(taggingClient Client) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&tagHandler{taggingClient: taggingClient}),
			api.RoleCC),
		httputil.Get, httputil.Delete)
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

func (h *tagHandler) parseGETRequest(r *http.Request) (*TagGETRequest, error) {
	rd := &TagGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagHandler) serveGET(w http.ResponseWriter, r *http.Request, req *TagGETRequest) {
	text := make([]string, 0, len(req.Text))
	for _, v := range req.Text {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			text = append(text, trimmed)
		}
	}

	if len(text) == 0 && !req.Common {
		httputil.JSONResponse(w, http.StatusOK, &TagGETResponse{
			Tags: []*response.Tag{},
		})
		return
	}

	ops := TONone
	if req.Common {
		ops = TOCommonOnly
	}
	tags, err := h.taggingClient.TagsFromText(text, ops)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &TagGETResponse{
		Tags: tags,
	})
}

func (h *tagHandler) parseDELETERequest(r *http.Request) (*TagDELETERequest, error) {
	rd := &TagDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *tagHandler) serveDELETE(w http.ResponseWriter, r *http.Request, req *TagDELETERequest) {
	if _, err := h.taggingClient.DeleteTag(req.ID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/tagging/model"
	"github.com/sprucehealth/backend/tagging/query"
	"github.com/sprucehealth/backend/tagging/response"
	"github.com/sprucehealth/backend/www"

	"github.com/sprucehealth/backend/libs/httputil"
)

type tagSavedSearchsHandler struct {
	taggingClient tagging.Client
}

type tagSavedSearchsGETResponse struct {
	SavedSearches []*response.TagSavedSearch `json:"saved_searches"`
}

type tagSavedSearchsPOSTRequest struct {
	Title string `json:"title"`
	Query string `json:"query"`
}

type tagSavedSearchsDELETERequest struct {
	ID int64 `schema:"id,required"`
}

func NewTagSavedSearchesHandler(taggingClient tagging.Client) http.Handler {
	return httputil.SupportedMethods(&tagSavedSearchsHandler{taggingClient: taggingClient}, httputil.Get, httputil.Post)
}

func (h *tagSavedSearchsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.serveGET(w, r)
	case "POST":
		req, err := h.parsePOSTRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePOST(w, r, req)
	}
}

func (h *tagSavedSearchsHandler) serveGET(w http.ResponseWriter, r *http.Request) {
	savedSearches, err := h.taggingClient.TagSavedSearchs()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	ssResponses := make([]*response.TagSavedSearch, len(savedSearches))
	for i, ss := range savedSearches {
		ssResponses[i] = response.TransformTagSavedSearch(ss)
	}

	httputil.JSONResponse(w, http.StatusOK, &tagSavedSearchsGETResponse{
		SavedSearches: ssResponses,
	})
}

func (h *tagSavedSearchsHandler) parsePOSTRequest(r *http.Request) (*tagSavedSearchsPOSTRequest, error) {
	rd := &tagSavedSearchsPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.Query == "" || rd.Title == "" {
		return nil, errors.New("query, title required")
	}

	// Validate that the query will parse correctly
	if _, err := query.NewTagAssociationQuery(rd.Query); query.IsErrBadExpression(err) {
		return nil, err
	}

	return rd, nil
}

func (h *tagSavedSearchsHandler) servePOST(w http.ResponseWriter, r *http.Request, req *tagSavedSearchsPOSTRequest) {
	if _, err := h.taggingClient.InsertTagSavedSearch(&model.TagSavedSearch{
		Title: req.Title,
		Query: req.Query,
	}); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

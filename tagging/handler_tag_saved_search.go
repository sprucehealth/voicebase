package tagging

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/tagging/response"

	"github.com/sprucehealth/backend/libs/httputil"
)

type tagSavedSearchHandler struct {
	taggingClient Client
}

type tagSavedSearchGETResponse struct {
	SavedSearches []*response.TagSavedSearch `json:"saved_searches"`
}

func NewTagSavedSearchHandler(taggingClient Client) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&tagSavedSearchHandler{taggingClient: taggingClient}),
		httputil.Get)
}

func (p *tagSavedSearchHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.RoleCC {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (h *tagSavedSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.serveGET(w, r)
	}
}

func (h *tagSavedSearchHandler) serveGET(w http.ResponseWriter, r *http.Request) {
	savedSearches, err := h.taggingClient.TagSavedSearchs()
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	ssResponses := make([]*response.TagSavedSearch, len(savedSearches))
	for i, ss := range savedSearches {
		ssResponses[i] = response.TransformTagSavedSearch(ss)
	}

	httputil.JSONResponse(w, http.StatusOK, &tagSavedSearchGETResponse{
		SavedSearches: ssResponses,
	})
}

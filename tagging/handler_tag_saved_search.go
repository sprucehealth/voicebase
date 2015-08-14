package tagging

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/tagging/response"
	"golang.org/x/net/context"
)

type tagSavedSearchHandler struct {
	taggingClient Client
}

type tagSavedSearchGETResponse struct {
	SavedSearches []*response.TagSavedSearch `json:"saved_searches"`
}

func NewTagSavedSearchHandler(taggingClient Client) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&tagSavedSearchHandler{taggingClient: taggingClient}),
			api.RoleCC),
		httputil.Get)
}

func (h *tagSavedSearchHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.serveGET(ctx, w, r)
	}
}

func (h *tagSavedSearchHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	savedSearches, err := h.taggingClient.TagSavedSearchs()
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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

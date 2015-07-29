package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/www"
)

type tagSavedSearchHandler struct {
	taggingClient tagging.Client
}

func newTagSavedSearchHandler(taggingClient tagging.Client) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&tagSavedSearchHandler{taggingClient: taggingClient}, httputil.Delete)
}

func (h *tagSavedSearchHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	switch r.Method {
	case "DELETE":
		h.serveDELETE(w, r, id)
	}
}

func (h *tagSavedSearchHandler) serveDELETE(w http.ResponseWriter, r *http.Request, id int64) {
	if _, err := h.taggingClient.DeleteTagSavedSearch(id); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

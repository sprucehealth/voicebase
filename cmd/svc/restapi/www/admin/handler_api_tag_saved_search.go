package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/tagging"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
)

type tagSavedSearchHandler struct {
	taggingClient tagging.Client
}

func newTagSavedSearchHandler(taggingClient tagging.Client) http.Handler {
	return httputil.SupportedMethods(&tagSavedSearchHandler{taggingClient: taggingClient}, httputil.Delete)
}

func (h *tagSavedSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r.Context())["id"], 10, 64)
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

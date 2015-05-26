package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/www"

	"github.com/sprucehealth/backend/libs/httputil"
)

type tagSavedSearchHandler struct {
	taggingClient tagging.Client
}

type tagSavedSearchDELETERequest struct {
	ID int64
}

func NewTagSavedSearchHandler(taggingClient tagging.Client) http.Handler {
	return httputil.SupportedMethods(&tagSavedSearchHandler{taggingClient: taggingClient}, httputil.Delete)
}

func (h *tagSavedSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		fmt.Println(err)
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

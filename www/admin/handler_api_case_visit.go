package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
)

type caseVisitHandler struct {
	dataAPI api.DataAPI
}

type caseVisitGETResponse struct {
	VisitSummary *responses.PHISafeVisitSummary `json:"visit_summary"`
}

func NewCaseVisitHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&caseVisitHandler{dataAPI: dataAPI}, []string{"GET"})
}

func (h *caseVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	visitID, err := strconv.ParseInt(mux.Vars(r)["visitID"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	switch r.Method {
	case "GET":
		h.serveGET(w, r, visitID)
	}
}

func (h *caseVisitHandler) serveGET(w http.ResponseWriter, r *http.Request, visitID int64) {
	summary, err := h.dataAPI.VisitSummary(visitID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	phiSafeSummary := responses.TransformVisitSummary(summary)

	httputil.JSONResponse(w, http.StatusOK, caseVisitGETResponse{
		VisitSummary: phiSafeSummary,
	})
}

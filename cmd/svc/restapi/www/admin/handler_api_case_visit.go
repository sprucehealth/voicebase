package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/responses"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/mux"
)

type caseVisitHandler struct {
	dataAPI api.DataAPI
}

type caseVisitGETResponse struct {
	VisitSummary *responses.PHISafeVisitSummary `json:"visit_summary"`
}

func newCaseVisitHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&caseVisitHandler{dataAPI: dataAPI}, httputil.Get)
}

func (h *caseVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	visitID, err := strconv.ParseInt(mux.Vars(r.Context())["visitID"], 10, 64)
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

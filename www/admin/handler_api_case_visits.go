package admin

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
)

const (
	VisitStatusUncompleted = "uncompleted"
)

type caseVisitsHandler struct {
	dataAPI api.DataAPI
}

type caseVisitsGETRequest struct {
	Status string `schema:"status,required"`
}

type caseVisitsGETResponse struct {
	VisitSummaries []*responses.PHISafeVisitSummary `json:"visit_summaries"`
}

func NewCaseVisitsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&caseVisitsHandler{dataAPI: dataAPI}, []string{"GET"})
}

func (h *caseVisitsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rd, err := h.parseGETRequest(w, r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveGET(w, r, rd)
	}
}

func (h *caseVisitsHandler) parseGETRequest(w http.ResponseWriter, r *http.Request) (*caseVisitsGETRequest, error) {
	rd := &caseVisitsGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *caseVisitsHandler) serveGET(w http.ResponseWriter, r *http.Request, rd *caseVisitsGETRequest) {
	var includedStatuses []string
	switch {
	case rd.Status == VisitStatusUncompleted:
		includedStatuses = []string{common.PVStatusRouted, common.PVStatusCharged, common.PVStatusReviewing, common.PVStatusSubmitted}
	default:
		www.APIBadRequestError(w, r, fmt.Sprintf("Unknown status for querying case visits - %s", rd.Status))
		return
	}

	summaries, err := h.dataAPI.VisitSummaries(includedStatuses)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	sort.Sort(common.ByVisitSummaryCreationDate(summaries))

	phiSafeSummaries := make([]*responses.PHISafeVisitSummary, len(summaries))
	for i, v := range summaries {
		phiSafeSummaries[i] = responses.TransformVisitSummary(v)
	}

	httputil.JSONResponse(w, http.StatusOK, caseVisitsGETResponse{
		VisitSummaries: phiSafeSummaries,
	})
}

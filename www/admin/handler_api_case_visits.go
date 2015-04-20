package admin

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
)

const (
	VisitStatusUncompleted = "uncompleted"
	VisitStatusSubmitted   = "submitted"
)

type caseVisitsHandler struct {
	dataAPI api.DataAPI
}

type caseVisitsGETRequest struct {
	Status   string   `schema:"status"`
	Statuses []string `schema:"statuses"`
	To       int64    `schema:"from"`
	From     int64    `schema:"to"`
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

	if rd.From != 0 && rd.To == 0 || rd.To != 0 && rd.From == 0 {
		return nil, fmt.Errorf("Unbounded time queries not allowed")
	}

	return rd, nil
}

func (h *caseVisitsHandler) serveGET(w http.ResponseWriter, r *http.Request, rd *caseVisitsGETRequest) {
	var includedStatuses []string
	switch {
	case rd.Status == VisitStatusUncompleted:
		includedStatuses = []string{common.PVStatusRouted, common.PVStatusCharged, common.PVStatusReviewing, common.PVStatusSubmitted}
	case rd.Status == VisitStatusSubmitted:
		includedStatuses = []string{common.PVStatusRouted, common.PVStatusCharged, common.PVStatusReviewing, common.PVStatusSubmitted, common.PVStatusTreated}
	}
	includedStatuses = append(includedStatuses, rd.Statuses...)

	var from time.Time
	var to time.Time
	if rd.From != 0 {
		from = time.Unix(rd.From, 0)
	}
	if rd.To != 0 {
		to = time.Unix(rd.To, 0)
	}
	summaries, err := h.dataAPI.VisitSummaries(includedStatuses, from, to)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	sort.Sort(common.ByVisitSummarySubmissionDate(summaries))

	phiSafeSummaries := make([]*responses.PHISafeVisitSummary, len(summaries))
	for i, v := range summaries {
		phiSafeSummaries[i] = responses.TransformVisitSummary(v)
	}

	httputil.JSONResponse(w, http.StatusOK, caseVisitsGETResponse{
		VisitSummaries: phiSafeSummaries,
	})
}

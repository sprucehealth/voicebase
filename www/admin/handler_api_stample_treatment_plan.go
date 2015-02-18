package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type stpHandler struct {
	dataAPI api.DataAPI
}

type stpGETRequest struct {
	PathwayTag string `schema:"pathway_tag,required"`
}

type stpPUTRequest struct {
	PathwayTag          string          `json:"pathway_tag"`
	SampleTreatmentPlan json.RawMessage `json:"sample_treatment_plan"`
}

func NewSampleTreatmentPlanHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&stpHandler{dataAPI: dataAPI}, []string{"GET", "PUT"})
}

func (h *stpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		requestData, err := h.parseGETRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveGET(w, r, requestData)
	case "PUT":
		requestData, err := h.parsePUTRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePUT(w, r, requestData)
	}
}

func (h *stpHandler) parseGETRequest(r *http.Request) (*stpGETRequest, error) {
	rd := &stpGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *stpHandler) parsePUTRequest(r *http.Request) (*stpPUTRequest, error) {
	rd := &stpPUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.PathwayTag == "" {
		return nil, fmt.Errorf("Incomplete request body - pathway_tag required")
	}

	if rd.SampleTreatmentPlan == nil {
		return nil, fmt.Errorf("Incomplete request body - sample_treatment_plan required")
	}

	return rd, nil
}

func (h *stpHandler) serveGET(w http.ResponseWriter, r *http.Request, req *stpGETRequest) {
	stp, err := h.dataAPI.PathwaySTP(req.PathwayTag)
	if err != nil && !api.IsErrNotFound(err) {
		return
	}

	var response interface{}
	if len(stp) > 0 {
		if err := json.Unmarshal(stp, &response); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}

func (h *stpHandler) servePUT(w http.ResponseWriter, r *http.Request, req *stpPUTRequest) {
	if err := h.dataAPI.CreatePathwaySTP(req.PathwayTag, req.SampleTreatmentPlan); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, nil)
}

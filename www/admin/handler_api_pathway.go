package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type pathwayHandler struct {
	dataAPI api.DataAPI
}

type pathwayResponse struct {
	Pathway *common.Pathway `json:"pathway"`
}

type updatePathwayRequest struct {
	Details *common.PathwayDetails `json:"details"`
}

func NewPathwayHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&pathwayHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "PUT"})
}

func (h *pathwayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
	case "PUT":
		h.put(w, r)
	}
}

func (h *pathwayHandler) get(w http.ResponseWriter, r *http.Request) {
	pathwayID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	pathway, err := h.dataAPI.Pathway(pathwayID, api.POWithDetails)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "GetPathway", map[string]interface{}{"pathway_id": pathwayID})

	if pathway.Details == nil {
		// Return empty details rather than null to serve as an
		// example of what fields are available.
		pathway.Details = &common.PathwayDetails{
			FAQ: []common.FAQ{
				{
					Question: "",
					Answer:   "",
				},
			},
			DidYouKnow:     []string{},
			WhatIsIncluded: []string{},
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &pathwayResponse{
		Pathway: pathway,
	})
}

func (h *pathwayHandler) put(w http.ResponseWriter, r *http.Request) {
	pathwayID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	var req updatePathwayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, "Failed to decode JSON body")
		return
	}

	pathway, err := h.dataAPI.Pathway(pathwayID, api.PONone)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	}
	pathway.Details = req.Details

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "UpdatePathway", map[string]interface{}{"pathway_id": pathwayID})

	if err := h.dataAPI.UpdatePathway(pathwayID, req.Details); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &pathwayResponse{
		Pathway: pathway,
	})
}

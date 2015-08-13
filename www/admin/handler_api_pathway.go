package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/www"
)

type pathwayHandler struct {
	dataAPI api.DataAPI
}

type pathwayResponse struct {
	Pathway *common.Pathway `json:"pathway"`
}

type updatePathwayRequest struct {
	Name    *string                `json:"name"`
	Details *common.PathwayDetails `json:"details"`
}

// NewPathwayHandler returns an HTTP handler that supports GET for requesting
// pathway details and PATCH for updating the pathway details.
func newPathwayHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&pathwayHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Patch)
}

func (h *pathwayHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		h.get(ctx, w, r)
	case httputil.Patch:
		h.patch(ctx, w, r)
	}
}

func (h *pathwayHandler) get(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	pathwayID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	pathway, err := h.dataAPI.Pathway(pathwayID, api.POWithDetails)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(ctx)
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
			AgeRestrictions: []*common.PathwayAgeRestriction{
				{
					VisitAllowed:  false,
					MaxAgeOfRange: ptr.Int(17),
					Alert: &common.PathwayAlert{
						Type:        "age_alert:error",
						Message:     "Sorry, we don't currently support the chosen condition for patients under 18.",
						ButtonTitle: "OK",
					},
				},
				{VisitAllowed: true},
			},
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &pathwayResponse{
		Pathway: pathway,
	})
}

func (h *pathwayHandler) patch(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	pathwayID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
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

	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "UpdatePathway", map[string]interface{}{"pathway_id": pathwayID})

	update := &api.PathwayUpdate{
		Name:    req.Name,
		Details: req.Details,
	}
	if ok, reason := req.Details.Validate(); !ok {
		www.APIBadRequestError(w, r, reason)
		return
	}
	if err := h.dataAPI.UpdatePathway(pathwayID, update); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &pathwayResponse{
		Pathway: pathway,
	})
}

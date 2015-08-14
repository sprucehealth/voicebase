package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type pathwaysListHandler struct {
	dataAPI api.DataAPI
}

type pathwaysListResponse struct {
	Pathways []*common.Pathway `json:"pathways"`
}

type createPathwayRequest struct {
	Pathway *common.Pathway `json:"pathway"`
}

func newPathwaysListHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&pathwaysListHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Post)
}

func (h *pathwaysListHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(ctx, w, r)
	case "POST":
		h.post(ctx, w, r)
	}
}

func (h *pathwaysListHandler) get(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "GetPathwayList", nil)

	var activeOnly bool
	if s := r.FormValue("active_only"); s != "" {
		var err error
		activeOnly, err = strconv.ParseBool(s)
		if err != nil {
			www.APIBadRequestError(w, r, "failed to parse active_only")
			return
		}
	}

	opts := api.PONone
	if activeOnly {
		opts |= api.POActiveOnly
	}

	pathways, err := h.dataAPI.ListPathways(opts)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &pathwaysListResponse{Pathways: pathways})
}

func (h *pathwaysListHandler) post(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "CreatePathway", nil)

	var req createPathwayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, "failed to decode json request body")
		return
	}

	req.Pathway.Status = common.PathwayActive
	if err := h.dataAPI.CreatePathway(req.Pathway); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	// create sku items for each pathway created
	_, err := h.dataAPI.CreateSKU(&common.SKU{
		Type:         req.Pathway.Tag + "_" + common.SCVisit.String(),
		CategoryType: common.SCVisit,
	})
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	_, err = h.dataAPI.CreateSKU(&common.SKU{
		Type:         req.Pathway.Tag + "_" + common.SCFollowup.String(),
		CategoryType: common.SCFollowup,
	})

	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &pathwayResponse{Pathway: req.Pathway})
}

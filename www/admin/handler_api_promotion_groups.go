package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type promotionGroupsHandler struct {
	dataAPI api.DataAPI
}

// PromotionGroupsGETResponse represents the data returned by sucessful GET requests to promotionHandler
type PromotionGroupsGETResponse struct {
	PromotionGroups []*responses.PromotionGroup `json:"promotion_groups"`
}

// newPromotionGroupsHandler returns a new initialized instance of promotionGroupsHandler
func newPromotionGroupsHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&promotionGroupsHandler{dataAPI: dataAPI}, httputil.Get)
}

func (h *promotionGroupsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		h.serveGET(ctx, w, r)
	}
}

func (h *promotionGroupsHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	promotionGroups, err := h.dataAPI.PromotionGroups()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	resps := make([]*responses.PromotionGroup, len(promotionGroups))
	for i, v := range promotionGroups {
		resps[i] = responses.TransformPromotionGroup(v)
	}
	httputil.JSONResponse(w, http.StatusOK, &PromotionGroupsGETResponse{PromotionGroups: resps})
}

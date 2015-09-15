package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"
	"github.com/sprucehealth/schema"
	"golang.org/x/net/context"
)

type promotionsHandler struct {
	dataAPI api.DataAPI
}

// PromotionsGETRequest represents the data expected to associated with a successful GET request
type PromotionsGETRequest struct {
	Types []string `schema:"type"`
}

// PromotionsGETResponse represents the data expected to returned from a successful GET request
type PromotionsGETResponse struct {
	Promotions []*responses.Promotion `json:"promotions"`
}

// PromotionsPOSTRequest represents the data expected to be associated with a successful POST request
type PromotionsPOSTRequest struct {
	Code          string `json:"code"`
	DataJSON      string `json:"data_json"`
	PromotionType string `json:"promo_type"`
	Group         string `json:"group"`
	Expires       *int64 `json:"expires"`
}

// PromotionsPOSTResponse represents the data expected to be returned from a successful POST request
type PromotionsPOSTResponse struct {
	PromoCodeID int64 `json:"promotion_code_id,string"`
}

// NewPromotionsHandler returns an initialized instance of promotionHandler
func newPromotionsHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&promotionsHandler{dataAPI: dataAPI}, httputil.Get, httputil.Post)
}

func (h *promotionsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		req, err := h.parseGETRequest(ctx, r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveGET(ctx, w, r, req)
	case httputil.Post:
		req, err := h.parsePOSTRequest(ctx, r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePOST(ctx, w, r, req)
	}
}

func (h *promotionsHandler) parseGETRequest(ctx context.Context, r *http.Request) (*PromotionsGETRequest, error) {
	rd := &PromotionsGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *promotionsHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, req *PromotionsGETRequest) {
	promotions, err := h.dataAPI.Promotions(nil, req.Types, common.PromotionTypes)
	if api.IsErrNotFound(err) {
		httputil.JSONResponse(w, http.StatusOK, &PromotionsGETResponse{Promotions: []*responses.Promotion{}})
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	resps := make([]*responses.Promotion, len(promotions))
	for i, v := range promotions {
		resps[i] = responses.TransformPromotion(v)
	}
	httputil.JSONResponse(w, http.StatusOK, &PromotionsGETResponse{Promotions: resps})
}

func (h *promotionsHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*PromotionsPOSTRequest, error) {
	rd := &PromotionsPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.Code == "" || rd.DataJSON == "" || rd.PromotionType == "" || rd.Group == "" {
		return nil, errors.New("code, data_json, promo_type, group required")
	}
	return rd, nil
}

func (h *promotionsHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, req *PromotionsPOSTRequest) {
	// Check if the code already exists
	if _, err := h.dataAPI.LookupPromoCode(req.Code); !api.IsErrNotFound(err) {
		www.APIBadRequestError(w, r, fmt.Sprintf("PromoCode %q is already in use by another promotion.", req.Code))
		return
	}

	promo := &common.Promotion{
		Code:  req.Code,
		Group: req.Group,
	}

	if req.Expires != nil {
		t := time.Unix(*req.Expires, 0)
		promo.Expires = &t
	}
	promotionDataType, ok := common.PromotionTypes[req.PromotionType]
	if !ok {
		www.APIBadRequestError(w, r, fmt.Sprintf("Unable to find promotion type: %s", req.PromotionType))
		return
	}

	promo.Data = reflect.New(promotionDataType).Interface().(common.Typed)
	if err := json.Unmarshal([]byte(req.DataJSON), &promo.Data); err != nil {
		www.APIBadRequestError(w, r, fmt.Sprintf("Unable to parse promotion data: %s - %v", req.DataJSON, err))
		return
	}

	promoCodeID, err := h.dataAPI.CreatePromotion(promo)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &PromotionsPOSTResponse{PromoCodeID: promoCodeID})
}

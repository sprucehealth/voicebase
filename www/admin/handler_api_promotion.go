package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/responses"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type promotionHandler struct {
	dataAPI api.DataAPI
}

type PromotionGETRequest struct {
	Types []string `schema:"type"`
}

type PromotionGETResponse struct {
	Promotions []*responses.Promotion `json:"promotions"`
}

type PromotionPOSTRequest struct {
	Code          string `json:"code"`
	DataJSON      string `json:"data_json"`
	PromotionType string `json:"promo_type"`
	Group         string `json:"group"`
	Expires       *int64 `json:"expires"`
}

type PromotionPOSTResponse struct {
	PromoCodeID int64 `json:"promotion_code_id"`
}

func NewPromotionHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&promotionHandler{dataAPI: dataAPI}, httputil.Get, httputil.Post)
}

func (h *promotionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		req, err := h.parseGETRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveGET(w, r, req)
	case "POST":
		req, err := h.parsePOSTRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePOST(w, r, req)
	}
}

func (h *promotionHandler) parseGETRequest(r *http.Request) (*PromotionGETRequest, error) {
	rd := &PromotionGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *promotionHandler) serveGET(w http.ResponseWriter, r *http.Request, req *PromotionGETRequest) {
	promotions, err := h.dataAPI.Promotions(nil, req.Types, common.PromotionTypes)
	if api.IsErrNotFound(err) {
		httputil.JSONResponse(w, http.StatusOK, &PromotionGETResponse{Promotions: []*responses.Promotion{}})
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	resps := make([]*responses.Promotion, len(promotions))
	for i, v := range promotions {
		resps[i] = responses.TransformPromotion(v)
	}
	httputil.JSONResponse(w, http.StatusOK, &PromotionGETResponse{Promotions: resps})
}

func (h *promotionHandler) parsePOSTRequest(r *http.Request) (*PromotionPOSTRequest, error) {
	rd := &PromotionPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.Code == "" || rd.DataJSON == "" || rd.PromotionType == "" || rd.Group == "" {
		return nil, errors.New("code, data_json, promo_type, group required")
	}
	return rd, nil
}

func (h *promotionHandler) servePOST(w http.ResponseWriter, r *http.Request, req *PromotionPOSTRequest) {
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
	httputil.JSONResponse(w, http.StatusOK, &PromotionPOSTResponse{PromoCodeID: promoCodeID})
}

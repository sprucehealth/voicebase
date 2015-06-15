package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/responses"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type promotionReferralRoutesHandler struct {
	dataAPI api.DataAPI
}

type PromotionReferralRoutesGETRequest struct {
	Lifecycles []string `schema:"lifecycles,required"`
}

type PromotionReferralRoutesGETResponse struct {
	PromotionReferralRoutes []*responses.PromotionReferralRoute `json:"promotion_referral_routes"`
}

type PromotionReferralRoutesPOSTRequest struct {
	PromotionCodeID int64   `json:"promotion_code_id"`
	Priority        int     `json:"priority"`
	Lifecycle       string  `json:"lifecycle"`
	Gender          *string `json:"gender"`
	AgeLower        *int    `json:"age_lower"`
	AgeUpper        *int    `json:"age_upper"`
	State           *string `json:"state"`
	Pharmacy        *string `json:"pharmacy"`
}

func NewPromotionReferralRoutesHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&promotionReferralRoutesHandler{dataAPI: dataAPI}, httputil.Get, httputil.Post)
}

func (h *promotionReferralRoutesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (h *promotionReferralRoutesHandler) parseGETRequest(r *http.Request) (*PromotionReferralRoutesGETRequest, error) {
	rd := &PromotionReferralRoutesGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *promotionReferralRoutesHandler) serveGET(w http.ResponseWriter, r *http.Request, req *PromotionReferralRoutesGETRequest) {
	routes, err := h.dataAPI.PromotionReferralRoutes(req.Lifecycles)
	if api.IsErrNotFound(err) {
		httputil.JSONResponse(w, http.StatusOK, &PromotionReferralRoutesGETResponse{PromotionReferralRoutes: []*responses.PromotionReferralRoute{}})
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	resps := make([]*responses.PromotionReferralRoute, len(routes))
	for i, v := range routes {
		resps[i] = responses.TransformPromotionReferralRoute(v)
	}
	httputil.JSONResponse(w, http.StatusOK, &PromotionReferralRoutesGETResponse{PromotionReferralRoutes: resps})
}

func (h *promotionReferralRoutesHandler) parsePOSTRequest(r *http.Request) (*PromotionReferralRoutesPOSTRequest, error) {
	rd := &PromotionReferralRoutesPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.PromotionCodeID == 0 || rd.Priority == 0 {
		return nil, errors.New("promotion_code_id, priority, lifecycle required")
	}
	return rd, nil
}

func (h *promotionReferralRoutesHandler) servePOST(w http.ResponseWriter, r *http.Request, req *PromotionReferralRoutesPOSTRequest) {
	var err error
	lifecycle, err := common.GetPRRLifecycle(req.Lifecycle)
	if err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	var gender *common.PRRGender
	if req.Gender != nil {
		prrg, err := common.GetPRRGender(*req.Gender)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		gender = &prrg
	}
	route := &common.PromotionReferralRoute{
		PromotionCodeID: req.PromotionCodeID,
		Priority:        req.Priority,
		Lifecycle:       lifecycle,
		Gender:          gender,
		AgeLower:        req.AgeLower,
		AgeUpper:        req.AgeUpper,
		State:           req.State,
		Pharmacy:        req.Pharmacy,
	}

	if _, err := h.dataAPI.InsertPromotionReferralRoute(route); err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

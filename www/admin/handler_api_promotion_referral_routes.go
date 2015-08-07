package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"
)

type promotionReferralRoutesHandler struct {
	dataAPI api.DataAPI
}

type promotionReferralRoutesGETRequest struct {
	Lifecycles []string `schema:"lifecycles,required"`
}

type promotionReferralRoutesGETResponse struct {
	PromotionReferralRoutes []*responses.PromotionReferralRoute `json:"promotion_referral_routes"`
}

type promotionReferralRoutesPOSTRequest struct {
	PromotionCodeID int64   `json:"promotion_code_id"`
	Priority        int     `json:"priority"`
	Lifecycle       string  `json:"lifecycle"`
	Gender          *string `json:"gender"`
	AgeLower        *int    `json:"age_lower"`
	AgeUpper        *int    `json:"age_upper"`
	State           *string `json:"state"`
	Pharmacy        *string `json:"pharmacy"`
}

func newPromotionReferralRoutesHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&promotionReferralRoutesHandler{dataAPI: dataAPI}, httputil.Get, httputil.Post)
}

func (h *promotionReferralRoutesHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

func (h *promotionReferralRoutesHandler) parseGETRequest(r *http.Request) (*promotionReferralRoutesGETRequest, error) {
	rd := &promotionReferralRoutesGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *promotionReferralRoutesHandler) serveGET(w http.ResponseWriter, r *http.Request, req *promotionReferralRoutesGETRequest) {
	routes, err := h.dataAPI.PromotionReferralRoutes(req.Lifecycles)
	if api.IsErrNotFound(err) {
		httputil.JSONResponse(w, http.StatusOK, &promotionReferralRoutesGETResponse{PromotionReferralRoutes: []*responses.PromotionReferralRoute{}})
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	resps := make([]*responses.PromotionReferralRoute, len(routes))
	for i, v := range routes {
		resps[i] = responses.TransformPromotionReferralRoute(v)
	}
	httputil.JSONResponse(w, http.StatusOK, &promotionReferralRoutesGETResponse{PromotionReferralRoutes: resps})
}

func (h *promotionReferralRoutesHandler) parsePOSTRequest(r *http.Request) (*promotionReferralRoutesPOSTRequest, error) {
	rd := &promotionReferralRoutesPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.PromotionCodeID == 0 || rd.Priority == 0 {
		return nil, errors.New("promotion_code_id, priority, lifecycle required")
	}
	return rd, nil
}

func (h *promotionReferralRoutesHandler) servePOST(w http.ResponseWriter, r *http.Request, req *promotionReferralRoutesPOSTRequest) {
	var err error
	lifecycle, err := common.ParsePRRLifecycle(req.Lifecycle)
	if err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	var gender *common.PRRGender
	if req.Gender != nil {
		prrg, err := common.ParsePRRGender(*req.Gender)
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

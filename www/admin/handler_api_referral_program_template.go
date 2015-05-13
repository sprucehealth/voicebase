package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/responses"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type referralProgramTemplateHandler struct {
	dataAPI api.DataAPI
}

type ReferralProgramTemplateGETRequest struct {
	Statuses common.ReferralProgramStatusList `json:"statuses"`
}

type ReferralProgramTemplateGETResponse struct {
	ReferralProgramTemplates []*responses.ReferralProgramTemplate `json:"referral_program_templates"`
}

type ReferralProgramTemplatePOSTRequest struct {
	PromotionCodeID int64                       `json:"promotion_code_id"`
	Title           string                      `json:"title"`
	Description     string                      `json:"description"`
	ShareText       *promotions.ShareTextParams `json:"share_text"`
	Group           string                      `json:"group"`
	HomeCard        *promotions.HomeCardConfig  `json:"home_card"`
}

type ReferralProgramTemplatePOSTResponse struct {
	ID int64 `json:"id"`
}

func NewReferralProgramTemplateHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&referralProgramTemplateHandler{dataAPI: dataAPI}, httputil.Get, httputil.Post)
}

func (h *referralProgramTemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (h *referralProgramTemplateHandler) parseGETRequest(r *http.Request) (*ReferralProgramTemplateGETRequest, error) {
	rd := &ReferralProgramTemplateGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *referralProgramTemplateHandler) serveGET(w http.ResponseWriter, r *http.Request, req *ReferralProgramTemplateGETRequest) {
	var err error
	templates, err := h.dataAPI.ReferralProgramTemplates(req.Statuses, common.PromotionTypes)
	if api.IsErrNotFound(err) {
		httputil.JSONResponse(w, http.StatusOK, &ReferralProgramTemplateGETResponse{ReferralProgramTemplates: []*responses.ReferralProgramTemplate{}})
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	resps := make([]*responses.ReferralProgramTemplate, len(templates))
	for i, v := range templates {
		resps[i] = responses.TransformReferralProgramTemplate(v)
	}
	httputil.JSONResponse(w, http.StatusOK, &ReferralProgramTemplateGETResponse{ReferralProgramTemplates: resps})
}

func (h *referralProgramTemplateHandler) parsePOSTRequest(r *http.Request) (*ReferralProgramTemplatePOSTRequest, error) {
	rd := &ReferralProgramTemplatePOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.PromotionCodeID == 0 || rd.Title == "" || rd.Description == "" || rd.Group == "" || rd.HomeCard == nil || rd.ShareText == nil {
		return nil, errors.New("promotion_code_id, title, description, share_text, group, home_card required")
	}
	return rd, nil
}

func (h *referralProgramTemplateHandler) servePOST(w http.ResponseWriter, r *http.Request, req *ReferralProgramTemplatePOSTRequest) {
	p, err := h.dataAPI.Promotion(req.PromotionCodeID, common.PromotionTypes)
	if api.IsErrNotFound(err) {
		www.APIBadRequestError(w, r, err.Error())
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	promotionData, ok := p.Data.(promotions.Promotion)
	if !ok {
		www.APIInternalError(w, r, err)
		return
	}

	if err := promotionData.Validate(); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	referralProgram, err := promotions.NewGiveReferralProgram(req.Title, req.Description, req.Group, req.HomeCard, promotionData, req.ShareText)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	referralProgramTemplate := &common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Data:            referralProgram,
		Status:          common.RSActive,
		PromotionCodeID: &req.PromotionCodeID,
	}

	id, err := h.dataAPI.CreateReferralProgramTemplate(referralProgramTemplate)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &ReferralProgramTemplatePOSTResponse{ID: id})
}

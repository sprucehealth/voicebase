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
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www"
)

type referralProgramTemplateHandler struct {
	dataAPI api.DataAPI
}

// ReferralProgramTemplateGETRequest represents the data expected in a sucessful GET request
type ReferralProgramTemplateGETRequest struct {
	Statuses common.ReferralProgramStatusList `json:"statuses"`
}

// ReferralProgramTemplateGETResponse represents the data returned in a sucessful GET request
type ReferralProgramTemplateGETResponse struct {
	ReferralProgramTemplates []*responses.ReferralProgramTemplate `json:"referral_program_templates"`
}

// ReferralProgramTemplatePOSTRequest represents the data expected in a sucessful POST request
type ReferralProgramTemplatePOSTRequest struct {
	PromotionCodeID int64                       `json:"promotion_code_id"`
	Title           string                      `json:"title"`
	Description     string                      `json:"description"`
	ShareText       *promotions.ShareTextParams `json:"share_text"`
	Group           string                      `json:"group"`
	HomeCard        *promotions.HomeCardConfig  `json:"home_card"`
	ImageURL        string                      `json:"image_url"`
	ImageWidth      int                         `json:"image_width"`
	ImageHeight     int                         `json:"image_height"`
}

// ReferralProgramTemplatePOSTResponse represents the data returned in a sucessful POST request
type ReferralProgramTemplatePOSTResponse struct {
	ID int64 `json:"id,string"`
}

// ReferralProgramTemplatePUTRequest represents the data expected in a sucessful PUT request
type ReferralProgramTemplatePUTRequest struct {
	ID     int64  `json:"id,string"`
	Status string `json:"status"`
}

// NewReferralProgramTemplateHandler returns an initialized instance of referralProgramTemplateHandler
func newReferralProgramTemplateHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&referralProgramTemplateHandler{dataAPI: dataAPI}, httputil.Get, httputil.Post, httputil.Put)
}

func (h *referralProgramTemplateHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rd, err := h.parseGETRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveGET(w, r, rd)
	case "POST":
		rd, err := h.parsePOSTRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePOST(w, r, rd)
	case "PUT":
		rd, err := h.parsePUTRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePUT(w, r, rd)
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

func (h *referralProgramTemplateHandler) serveGET(w http.ResponseWriter, r *http.Request, rd *ReferralProgramTemplateGETRequest) {
	var err error
	templates, err := h.dataAPI.ReferralProgramTemplates(rd.Statuses, common.PromotionTypes)
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

func (h *referralProgramTemplateHandler) servePOST(w http.ResponseWriter, r *http.Request, rd *ReferralProgramTemplatePOSTRequest) {
	p, err := h.dataAPI.Promotion(rd.PromotionCodeID, common.PromotionTypes)
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

	referralProgram, err := promotions.NewGiveReferralProgram(rd.Title, rd.Description, rd.Group, rd.HomeCard, promotionData, rd.ShareText, rd.ImageURL, rd.ImageWidth, rd.ImageHeight)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	referralProgramTemplate := &common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Data:            referralProgram,
		Status:          common.RSActive,
		PromotionCodeID: &rd.PromotionCodeID,
	}

	id, err := h.dataAPI.CreateReferralProgramTemplate(referralProgramTemplate)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &ReferralProgramTemplatePOSTResponse{ID: id})
}

func (h *referralProgramTemplateHandler) parsePUTRequest(r *http.Request) (*ReferralProgramTemplatePUTRequest, error) {
	rd := &ReferralProgramTemplatePUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.Status == "" || rd.ID == 0 {
		return nil, errors.New("id, status required")
	}

	return rd, nil
}

func (h *referralProgramTemplateHandler) servePUT(w http.ResponseWriter, r *http.Request, rd *ReferralProgramTemplatePUTRequest) {
	rps, err := common.ParseReferralProgramStatus(rd.Status)
	if err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	if rps != common.RSDefault {
		rpt, err := h.dataAPI.ReferralProgramTemplate(rd.ID, common.PromotionTypes)
		if api.IsErrNotFound(err) {
			www.APIBadRequestError(w, r, err.Error())
			return
		} else if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		if rpt.Status == common.RSDefault {
			www.APIBadRequestError(w, r, "The Default Referral Program Template cannot have it's status modified.")
			return
		}
	}

	if rps == common.RSDefault {
		if err := h.dataAPI.SetDefaultReferralProgramTemplate(rd.ID); api.IsErrNotFound(err) {
			www.APIBadRequestError(w, r, err.Error())
			return
		} else if err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	} else if rps == common.RSInactive {
		if err := h.dataAPI.InactivateReferralProgramTemplate(rd.ID); api.IsErrNotFound(err) {
			www.APIBadRequestError(w, r, err.Error())
			return
		} else if err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	} else {
		aff, err := h.dataAPI.UpdateReferralProgramTemplate(&common.ReferralProgramTemplateUpdate{
			ID:     rd.ID,
			Status: rps,
		})
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		if aff == 0 {
			www.APIBadRequestError(w, r, api.ErrNotFound(`referral_program_template`).Error())
			return
		}
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

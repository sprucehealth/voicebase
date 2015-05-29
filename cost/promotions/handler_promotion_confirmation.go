package promotions

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type promotionConfirmationHandler struct {
	dataAPI api.DataAPI
}

type PromotionConfirmationGETRequest struct {
	Code string `schema:"code,required"`
}

type PromotionConfirmationGETResponse struct {
	Title       string `json:"title"`
	ImageURL    string `json:"image_url"`
	BodyText    string `json:"body_text"`
	ButtonTitle string `json:"button_title"`
}

func NewPromotionConfirmationHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.NoAuthorizationRequired(
		httputil.SupportedMethods(&promotionConfirmationHandler{
			dataAPI: dataAPI,
		}, httputil.Get))
}

func (p *promotionConfirmationHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (h *promotionConfirmationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		req, err := h.parseGETRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.serveGET(w, r, req)
	}
}

func (h *promotionConfirmationHandler) parseGETRequest(r *http.Request) (*PromotionConfirmationGETRequest, error) {
	rd := &PromotionConfirmationGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *promotionConfirmationHandler) serveGET(w http.ResponseWriter, r *http.Request, req *PromotionConfirmationGETRequest) {
	// Check if the code provided is an account_code. If so we need to get the active referral program for that account
	promoCode, err := h.dataAPI.LookupPromoCode(req.Code)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError(fmt.Sprintf("Unable to find promotion for code %s", req.Code), w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var p *common.Promotion
	title := "Congratulations!"
	if promoCode.IsReferral {
		rp, err := h.dataAPI.ReferralProgram(promoCode.ID, common.PromotionTypes)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		referralProgram := rp.Data.(ReferralProgram)
		p = referralProgram.PromotionForReferredAccount(promoCode.Code)

		// Promotion codes could come from doctors or patients. The most common should be patient so look that up first.
		// This information will help us construct the appropriate message
		patient, err := h.dataAPI.GetPatientFromAccountID(rp.AccountID)
		if err != nil && api.IsErrNotFound(err) {
			_, err := h.dataAPI.GetDoctorFromAccountID(rp.AccountID)
			if err != nil {
				apiservice.WriteError(fmt.Errorf("Unable to locate referral program owner for Account ID %d. Checked both patient and doctor records.", rp.AccountID), w, r)
				return
			}
			title = "Welcome to Spruce!"
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		} else {
			title = fmt.Sprintf("Your friend %s has given you a free visit.", patient.FirstName)
		}
	} else {
		p, err = h.dataAPI.Promotion(promoCode.ID, common.PromotionTypes)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	promotion, ok := p.Data.(Promotion)
	if !ok {
		apiservice.WriteError(errors.New("Unable to cast promotion data into Promotion type"), w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &PromotionConfirmationGETResponse{
		Title:       title,
		ImageURL:    "spruce:///image/icon_case_large",
		BodyText:    promotion.SuccessMessage(),
		ButtonTitle: "Let's Go",
	})
}

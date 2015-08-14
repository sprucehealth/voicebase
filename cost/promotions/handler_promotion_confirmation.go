package promotions

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type promotionConfirmationHandler struct {
	dataAPI         api.DataAPI
	analyticsLogger analytics.Logger
}

// PromotionConfirmationGETRequest represents the data expected to be sent to the promotionConfirmationHandler in a GET request, it is exported for client consumption.
type PromotionConfirmationGETRequest struct {
	Code string `schema:"code,required"`
}

// PromotionConfirmationGETResponse represents the data returned from a successful GET request to the promotionConfirmationHandler, it is exported for client consumption.
type PromotionConfirmationGETResponse struct {
	Title       string `json:"title"`
	ImageURL    string `json:"image_url"`
	BodyText    string `json:"body_text"`
	ButtonTitle string `json:"button_title"`
}

// NewPromotionConfirmationHandler returns a new instance of the promotionConfirmationHandler
func NewPromotionConfirmationHandler(dataAPI api.DataAPI, analyticsLogger analytics.Logger) httputil.ContextHandler {
	return apiservice.NoAuthorizationRequired(
		httputil.SupportedMethods(&promotionConfirmationHandler{
			dataAPI:         dataAPI,
			analyticsLogger: analyticsLogger,
		}, httputil.Get))
}

func (h *promotionConfirmationHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		req, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.serveGET(ctx, w, r, req)
	}
}

func (h *promotionConfirmationHandler) parseGETRequest(ctx context.Context, r *http.Request) (*PromotionConfirmationGETRequest, error) {
	rd := &PromotionConfirmationGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *promotionConfirmationHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, req *PromotionConfirmationGETRequest) {
	// Check if the code provided is an account_code. If so we need to get the active referral program for that account
	promoCode, err := h.dataAPI.LookupPromoCode(req.Code)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError(ctx, fmt.Sprintf("Unable to find promotion for code %s", req.Code), w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	var p *common.Promotion
	var title string
	code := promoCode.Code
	codeID := promoCode.ID
	if promoCode.IsReferral {
		rp, err := h.dataAPI.ReferralProgram(promoCode.ID, common.PromotionTypes)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
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
				apiservice.WriteError(ctx, fmt.Errorf("Unable to locate referral program owner for Account ID %d. Checked both patient and doctor records.", rp.AccountID), w, r)
				return
			}
			title = "Welcome to Spruce"
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		} else {
			title = fmt.Sprintf("Your friend %s has given you a free visit.", patient.FirstName)
		}

		if rp.TemplateID != nil {
			rpt, err := h.dataAPI.ReferralProgramTemplate(*rp.TemplateID, common.PromotionTypes)
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}
			if rpt.PromotionCodeID != nil {
				promotion, err := h.dataAPI.Promotion(*rpt.PromotionCodeID, common.PromotionTypes)
				if err != nil {
					apiservice.WriteError(ctx, err, w, r)
					return
				}
				code = promotion.Code
				codeID = promotion.CodeID
			}
		}

		h.analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "referral_code_install_confirmation",
				Timestamp: analytics.Time(time.Now()),
				AccountID: rp.AccountID,
				ExtraJSON: analytics.JSONString(struct {
					Code   string `json:"code"`
					CodeID int64  `json:"code_id"`
				}{
					Code:   code,
					CodeID: codeID,
				}),
			},
		})
	} else {
		p, err = h.dataAPI.Promotion(promoCode.ID, common.PromotionTypes)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	h.analyticsLogger.WriteEvents([]analytics.Event{
		&analytics.ServerEvent{
			Event:     "promo_code_install_confirmation",
			Timestamp: analytics.Time(time.Now()),
			ExtraJSON: analytics.JSONString(struct {
				Code   string `json:"code"`
				CodeID int64  `json:"code_id"`
			}{
				Code:   code,
				CodeID: codeID,
			}),
		},
	})

	promotion, ok := p.Data.(Promotion)
	if !ok {
		apiservice.WriteError(ctx, errors.New("Unable to cast promotion data into Promotion type"), w, r)
		return
	}

	// If the confirmation requested is from a promotion and not a referal populate the title accordingly
	if !promoCode.IsReferral {
		title = promotion.DisplayMessage()
	}

	imageURL := promotion.ImageURL()
	if imageURL == "" {
		imageURL = DefaultPromotionImageURL
	}

	httputil.JSONResponse(w, http.StatusOK, &PromotionConfirmationGETResponse{
		Title:       title,
		ImageURL:    imageURL,
		BodyText:    promotion.SuccessMessage(),
		ButtonTitle: "Let's Go",
	})
}

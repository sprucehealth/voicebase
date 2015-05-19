package promotions

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type promotionConfirmationHandler struct {
	dataAPI api.DataAPI
}

type promotionConfirmationGETRequest struct {
	Code string `schema:"code,required"`
}

type promotionConfirmationGETResponse struct {
	Title       string `json:"title"`
	ImageURL    string `json:"image_url"`
	BodyText    string `json:"body_text"`
	ButtonTitle string `json:"button_title"`
}

func NewPromotionConfirmationHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.NoAuthorizationRequired(
		httputil.SupportedMethods(&promotionConfirmationHandler{
			dataAPI: dataAPI,
		}, []string{"GET"}))
}

func (p *promotionConfirmationHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (h *promotionConfirmationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		req, err := h.parseGETRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.serveGET(w, r, req)
	}
}

func (h *promotionConfirmationHandler) parseGETRequest(r *http.Request) (*promotionConfirmationGETRequest, error) {
	rd := &promotionConfirmationGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *promotionConfirmationHandler) serveGET(w http.ResponseWriter, r *http.Request, req *promotionConfirmationGETRequest) {
	promoCode, err := h.dataAPI.LookupPromoCode(req.Code)
	if err != nil {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}

	p, err := h.dataAPI.Promotion(promoCode.ID, Types)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	promotion, ok := p.Data.(Promotion)
	if !ok {
		apiservice.WriteError(errors.New("Unable to cast promotion data into Promotion type"), w, r)
		return
	}

	title := "Congratulations!"
	if promoCode.IsReferral {
		rp, err := h.dataAPI.ReferralProgram(promoCode.ID, Types)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		patient, err := h.dataAPI.GetPatientFromAccountID(rp.AccountID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		title = fmt.Sprintf("Your friend %s has given you a free visit.", patient.FirstName)
	}

	httputil.JSONResponse(w, http.StatusOK, &promotionConfirmationGETResponse{
		Title:       title,
		ImageURL:    "spruce:///image/icon_case_large",
		BodyText:    promotion.SuccessMessage(),
		ButtonTitle: "Let's Go",
	})
}

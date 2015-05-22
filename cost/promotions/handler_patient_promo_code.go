package promotions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
)

type patientPromotionsHandler struct {
	dataAPI         api.DataAPI
	authAPI         api.AuthAPI
	analyticsLogger analytics.Logger
}

type PatientPromotionGETResponse struct {
	ActivePromotions  []*responses.Promotion `json:"active_promotions"`
	ExpiredPromotions []*responses.Promotion `json:"expired_promotions"`
}

type PatientPromotionPOSTRequest struct {
	PromoCode string `json:"promo_code"`
}

type PatientPromotionPOSTErrorResponse struct {
	UserError string `json:"user_error"`
	RequestID int64  `json:"request_id,string"`
}

func NewPatientPromotionsHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, analyticsLogger analytics.Logger) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(&patientPromotionsHandler{
				dataAPI:         dataAPI,
				authAPI:         authAPI,
				analyticsLogger: analyticsLogger,
			}),
			[]string{api.RolePatient}),
		httputil.Get, httputil.Post)
}

func (p *patientPromotionsHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (h *patientPromotionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.serveGET(w, r)
	case "POST":
		rd, err := h.parsePOSTRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.servePOST(w, r, rd)
	}
}

func (h *patientPromotionsHandler) serveGET(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	pendingPromotions, err := h.dataAPI.PendingPromotionsForAccount(ctxt.AccountID, Types)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var descSuffix string
	var containsTokens bool
	activePromotions := make([]*responses.Promotion, 0, len(pendingPromotions))
	expiredPromotions := make([]*responses.Promotion, 0, len(pendingPromotions))
	now := time.Now().Unix()
	for _, p := range pendingPromotions {
		promotion, ok := p.Data.(Promotion)
		if !ok {
			apiservice.WriteError(errors.New("Unable to cast promotion data into Promotion type"), w, r)
			return
		}
		var expireEpoch int64
		if p.Expires != nil {
			containsTokens = true
			descSuffix = "Expires <expiration_date>"
			expireEpoch = p.Expires.Unix()
		}
		if p.Expires != nil && (*p.Expires).Unix() < now {
			expiredPromotions = append(expiredPromotions, &responses.Promotion{
				Code:                 p.Code,
				Description:          promotion.SuccessMessage() + " Your discount will be applied at checkout. " + descSuffix,
				DescriptionHasTokens: containsTokens,
				ExpirationDate:       expireEpoch,
			})
		} else {
			activePromotions = append(activePromotions, &responses.Promotion{
				Code:                 p.Code,
				Description:          promotion.SuccessMessage() + " Your discount will be applied at checkout. " + descSuffix,
				DescriptionHasTokens: containsTokens,
				ExpirationDate:       expireEpoch,
			})
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &PatientPromotionGETResponse{
		ActivePromotions:  activePromotions,
		ExpiredPromotions: expiredPromotions,
	})
}

func (h *patientPromotionsHandler) parsePOSTRequest(r *http.Request) (*PatientPromotionPOSTRequest, error) {
	rd := &PatientPromotionPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *patientPromotionsHandler) servePOST(w http.ResponseWriter, r *http.Request, rd *PatientPromotionPOSTRequest) {
	ctxt := apiservice.GetContext(r)
	promoCode, err := h.dataAPI.LookupPromoCode(rd.PromoCode)
	if api.IsErrNotFound(err) {
		httputil.JSONResponse(w, http.StatusNotFound, &PatientPromotionPOSTErrorResponse{
			UserError: fmt.Sprintf("Sorry, the promo code %q is not valid.", rd.PromoCode),
			RequestID: ctxt.RequestID,
		})
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// If this isn't a referral code then check if the promotion is still active.
	if !promoCode.IsReferral {
		p, err := h.dataAPI.Promotion(promoCode.ID, Types)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if p.Expires != nil && (*p.Expires).Unix() < time.Now().Unix() {
			httputil.JSONResponse(w, http.StatusNotFound, &PatientPromotionPOSTErrorResponse{
				UserError: fmt.Sprintf("Sorry, the promo code %q is no longer active.", rd.PromoCode),
				RequestID: ctxt.RequestID,
			})
			return
		}
	}

	patient, err := h.dataAPI.GetPatientFromAccountID(ctxt.AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	_, state, err := h.dataAPI.PatientLocation(patient.PatientID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	accountPromotions, err := h.dataAPI.PendingPromotionsForAccount(ctxt.AccountID, Types)
	for _, ap := range accountPromotions {
		_, err := h.dataAPI.DeleteAccountPromotion(ap.AccountID, ap.CodeID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	// Associate the promo code then return it as if it was a get request. We know we are operating on the logged in account so perform this action synchronously
	async := false
	if _, err := AssociatePromoCode(patient.Email, state, rd.PromoCode, h.dataAPI, h.authAPI, h.analyticsLogger, async); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	h.serveGET(w, r)
}

package promotions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
)

type patientPromotionsHandler struct {
	dataAPI         api.DataAPI
	authAPI         api.AuthAPI
	analyticsLogger analytics.Logger
}

// PatientPromotionGETResponse represents the data returned from a successful GET request to the patientPromotionsHandler
type PatientPromotionGETResponse struct {
	ActivePromotions  []*responses.ClientPromotion `json:"active_promotions"`
	ExpiredPromotions []*responses.ClientPromotion `json:"expired_promotions"`
}

// PatientPromotionPOSTRequest represents the data expected to be sent to the patientPromotionsHandler in a POST request
type PatientPromotionPOSTRequest struct {
	PromoCode string `json:"promo_code"`
}

// PatientPromotionPOSTErrorResponse represents the data returned from a non standard POST request to the patientPromotionsHandler, it is exported for client consumption.
type PatientPromotionPOSTErrorResponse struct {
	UserError string `json:"user_error"`
	RequestID uint64 `json:"request_id,string"`
}

// NewPatientPromotionsHandler rreturns a new initialized instance of the patientPromotionsHandler
func NewPatientPromotionsHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, analyticsLogger analytics.Logger) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(&patientPromotionsHandler{
				dataAPI:         dataAPI,
				authAPI:         authAPI,
				analyticsLogger: analyticsLogger,
			}),
			api.RolePatient),
		httputil.Get, httputil.Post)
}

func (*patientPromotionsHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (h *patientPromotionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		h.serveGET(w, r)
	case httputil.Post:
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
	pendingPromotions, err := h.dataAPI.PendingPromotionsForAccount(ctxt.AccountID, common.PromotionTypes)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	sort.Sort(sort.Reverse(common.AccountPromotionByCreation(pendingPromotions)))

	var descSuffix string
	var containsTokens bool
	activePromotions := make([]*responses.ClientPromotion, 0, len(pendingPromotions))
	expiredPromotions := make([]*responses.ClientPromotion, 0, len(pendingPromotions))
	now := time.Now().Unix()
	for _, p := range pendingPromotions {
		promotion, ok := p.Data.(Promotion)
		if !ok {
			apiservice.WriteError(errors.New("Unable to cast promotion data into Promotion type"), w, r)
			return
		}

		// If we are listing promtions and the promotion has no value to the patient then ignore it
		if promotion.IsZeroValue() {
			continue
		}

		var expireEpoch int64
		if p.Expires != nil {
			containsTokens = true
			descSuffix = " - Expires <expiration_date>"
			expireEpoch = p.Expires.Unix()
		} else {
			containsTokens = false
			descSuffix = ""
		}
		if p.Expires != nil && (*p.Expires).Unix() < now {
			expiredPromotions = append(expiredPromotions, &responses.ClientPromotion{
				Code:                 p.Code,
				Description:          promotion.SuccessMessage() + descSuffix,
				DescriptionHasTokens: containsTokens,
				ExpirationDate:       expireEpoch,
			})
		} else {
			activePromotions = append(activePromotions, &responses.ClientPromotion{
				Code:                 p.Code,
				Description:          promotion.SuccessMessage() + descSuffix,
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

	if rd.PromoCode == "" {
		return nil, errors.New("promo_code required")
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
	var p *common.Promotion
	if !promoCode.IsReferral {
		p, err = h.dataAPI.Promotion(promoCode.ID, common.PromotionTypes)
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
	} else {
		arp, err := h.dataAPI.ActiveReferralProgramForAccount(ctxt.AccountID, common.PromotionTypes)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if arp.CodeID == promoCode.ID {
			httputil.JSONResponse(w, http.StatusNotFound, &PatientPromotionPOSTErrorResponse{
				UserError: fmt.Sprintf("%s has not been applied. A referral code cannot be claimed by the referrer ;)", rd.PromoCode),
				RequestID: ctxt.RequestID,
			})
			return
		}

		rp, err := h.dataAPI.ReferralProgram(promoCode.ID, common.PromotionTypes)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		referralProgram := rp.Data.(ReferralProgram)
		p = referralProgram.PromotionForReferredAccount(promoCode.Code)
	}

	patient, err := h.dataAPI.GetPatientFromAccountID(ctxt.AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	_, state, err := h.dataAPI.PatientLocation(patient.ID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	promotionGroup, err := h.dataAPI.PromotionGroup(p.Group)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	count, err := h.dataAPI.PromotionCountInGroupForAccount(ctxt.AccountID, p.Group)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// If the patient has reached their maximum for this promotion group then move the oldest unclaimed promo to the DELETED state
	// If this doesn't free up space then a failure should occur during AssociatePromoCode
	if promotionGroup.MaxAllowedPromos == count {
		accountPromotions, err := h.dataAPI.PendingPromotionsForAccount(ctxt.AccountID, common.PromotionTypes)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		for _, ap := range accountPromotions {
			if ap.GroupID == promotionGroup.ID {
				_, err := h.dataAPI.DeleteAccountPromotion(ap.AccountID, ap.CodeID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				break
			}
		}
	}

	// Associate the promo code then return it as if it was a get request. We know we are operating on the logged in account so perform this action synchronously
	async := false
	if _, err := AssociatePromoCode(patient.Email, state, rd.PromoCode, h.dataAPI, h.authAPI, h.analyticsLogger, async); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// If the promotion doesn't have any value for them then we don't want to return success and then have the GET return an empty list. This would be a confusing experience.
	// To fix this we will return a 404 here with a message explaining that it was applied before the empty screen is shown. Returning this error is the only way currently to display a message to the user.
	if p, ok := p.Data.(Promotion); ok && p.IsZeroValue() {
		httputil.JSONResponse(w, http.StatusNotFound, &PatientPromotionPOSTErrorResponse{
			UserError: fmt.Sprintf("The promo code %s has been associated with your account.", rd.PromoCode),
			RequestID: ctxt.RequestID,
		})
		return
	}

	h.serveGET(w, r)
}

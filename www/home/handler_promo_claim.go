package home

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/branch"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type refContext struct {
	Code       string
	IsReferral bool
	Title      string
	Message    string
}

type promoClaimHandler struct {
	dataAPI         api.DataAPI
	authAPI         api.AuthAPI
	branchClient    branch.Client
	analyticsLogger analytics.Logger
	refTemplate     *template.Template
}

func newPromoClaimHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, branchClient branch.Client, analyticsLogger analytics.Logger, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&promoClaimHandler{
		dataAPI:         dataAPI,
		authAPI:         authAPI,
		branchClient:    branchClient,
		analyticsLogger: analyticsLogger,
		refTemplate:     templateLoader.MustLoadTemplate("home/referral.html", "home/base.html", nil),
	}, httputil.Get, httputil.Post)
}

func (h *promoClaimHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code, err := h.dataAPI.LookupPromoCode(mux.Vars(r)["code"])
	if api.IsErrNotFound(err) {
		ctx := &refContext{
			Message: "Sorry, the promotion or referral code is no longer active.",
		}
		www.TemplateResponse(w, http.StatusOK, h.refTemplate, &www.BaseTemplateContext{
			Environment: environment.GetCurrent(),
			Title:       template.HTML("Claim a Promotion"),
			SubContext: &homeContext{
				NoBaseHeader: true,
				SubContext:   ctx,
			},
		})
		return
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	ctx := &refContext{
		Code:       code.Code,
		IsReferral: code.IsReferral,
	}

	if code.IsReferral {
		ref, err := h.dataAPI.ReferralProgram(code.ID, common.PromotionTypes)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		if ref == nil || ref.Status == common.RSInactive {
			ctx.Message = "Sorry, the referral code is no longer active."
			www.TemplateResponse(w, http.StatusOK, h.refTemplate, &www.BaseTemplateContext{
				Environment: environment.GetCurrent(),
				Title:       "Referral | Spruce",
				SubContext: &homeContext{
					SubContext: ctx,
				},
			})
			return
		}

		patient, err := h.dataAPI.GetPatientFromAccountID(ref.AccountID)
		if api.IsErrNotFound(err) {
			dr, err := h.dataAPI.GetDoctorFromAccountID(ref.AccountID)
			if api.IsErrNotFound(err) {
				www.InternalServerError(w, r, fmt.Errorf("neither doctor nor patient found for account ID %d", ref.AccountID))
				return
			} else if err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			ctx.Title = "Start a visit with " + dr.LongDisplayName + " on Spruce."
		} else if err != nil {
			www.InternalServerError(w, r, err)
			return
		} else {
			ctx.Title = "Your friend " + patient.FirstName + " has given you a free visit with a board-certified dermatologist."
		}
	} else {
		promo, err := promotions.LookupPromoCode(ctx.Code, h.dataAPI, h.analyticsLogger)
		if err == promotions.ErrPromotionExpired {
			promo = nil
		} else if err != promotions.ErrInvalidCode && err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		if promo == nil {
			ctx.Message = "Sorry, the referral code is no longer active."
			www.TemplateResponse(w, http.StatusOK, h.refTemplate, &www.BaseTemplateContext{
				Environment: environment.GetCurrent(),
				Title:       "Referral | Spruce",
				SubContext: &homeContext{
					SubContext: ctx,
				},
			})
			return
		}
		ctx.Title = promo.Title
	}

	// If page is being loaded from an iPhone or iPod touch then redirect to the branch link directly.
	if strings.Contains(r.UserAgent(), "iPhone") || strings.Contains(r.UserAgent(), "iPod") {
		earl, err := h.branchClient.URL(map[string]interface{}{
			"promo_code": code.Code,
			"source":     referralBranchSource,
		})
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		http.Redirect(w, r, earl, http.StatusFound)
		return
	}

	www.TemplateResponse(w, http.StatusOK, h.refTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       "Referral | Spruce",
		SubContext: &homeContext{
			SubContext: ctx,
		},
	})
}

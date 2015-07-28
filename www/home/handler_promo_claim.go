package home

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/branch"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

const (
	// PromoCodeKey represents the key associated with the branch link and url for the provided promo code
	PromoCodeKey = "promo_code"

	// SourceKey represent the key associated with the branch link and url for the provided install source
	SourceKey = "source"

	// ProviderIDKey represents the key associated with the branch link and id of the referring provider.
	ProviderIDKey = "provider_id"
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
	} else if err != nil && err != api.ErrValidAccountCodeNoActiveReferralProgram {
		www.InternalServerError(w, r, err)
		return
	}

	ctx := &refContext{}
	var providerID int64
	if code != nil {
		ctx.Code = code.Code
		ctx.IsReferral = code.IsReferral

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
				providerID = dr.ID.Int64()
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
	} else {
		ctx.Title = "Welcome to Spruce"
	}

	// If page is being loaded from an iPhone, iPod touch or android device, then redirect to the branch link directly.
	if strings.Contains(r.UserAgent(), "iPhone") || strings.Contains(r.UserAgent(), "iPod") || strings.Contains(strings.ToLower(r.UserAgent()), "android") {
		// Grab any parameters associated with our URL and throw them onto the branch link
		branchParams := map[string]interface{}{
			SourceKey: referralBranchSource,
		}

		if code != nil {
			branchParams[PromoCodeKey] = code.Code
		}

		if providerID > 0 {
			branchParams[ProviderIDKey] = strconv.FormatInt(providerID, 10)
		}

		if err := r.ParseForm(); err != nil {
			golog.Errorf("Failed to parse form for request %s, no failing request but params will not be provided to branch.", r.URL.String())
		}
		for k, v := range r.Form {
			if k == PromoCodeKey || k == SourceKey {
				golog.Infof("Not attaching URL query param %s:%s to branch link as %s is a managed param.", k, v, k)
			} else {
				if len(v) == 1 {
					branchParams[k] = v[0]
				} else if len(v) > 1 {
					branchParams[k] = v
				}
			}
		}

		earl, err := h.branchClient.URL(branchParams)
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

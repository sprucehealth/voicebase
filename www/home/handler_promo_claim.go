package home

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type promoContext struct {
	Code           string
	Email          string
	State          string
	StateName      string
	Platform       string
	States         []*common.State
	Promo          *promotions.PromotionDisplayInfo
	Claimed        bool
	InState        bool
	Message        string
	SuccessMessage string
	Android        bool
	Errors         map[string]string
}

type refContext struct {
	Message      string
	Ref          *common.ReferralProgram
	IsDoctor     bool
	ReferrerName string
}

type promoClaimHandler struct {
	dataAPI         api.DataAPI
	authAPI         api.AuthAPI
	analyticsLogger analytics.Logger
	promoTemplate   *template.Template
	refTemplate     *template.Template
	experimentID    string
}

func newPromoClaimHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, analyticsLogger analytics.Logger, templateLoader *www.TemplateLoader, experimentID string) http.Handler {
	return httputil.SupportedMethods(&promoClaimHandler{
		dataAPI:         dataAPI,
		authAPI:         authAPI,
		analyticsLogger: analyticsLogger,
		promoTemplate:   templateLoader.MustLoadTemplate("promotions/claim.html", "promotions/base.html", nil),
		refTemplate:     templateLoader.MustLoadTemplate("promotions/referral.html", "home/base.html", nil),
		experimentID:    experimentID,
	}, httputil.Get, httputil.Post)
}

func (h *promoClaimHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code, err := h.dataAPI.LookupPromoCode(mux.Vars(r)["code"])
	if api.IsErrNotFound(err) {
		ctx := &promoContext{
			Message: "Sorry, the promotion or referral code is no longer active.",
		}
		www.TemplateResponse(w, http.StatusOK, h.promoTemplate, &www.BaseTemplateContext{
			Environment: environment.GetCurrent(),
			Title:       template.HTML("Claim a Promotion"),
			SubContext: &homeContext{
				NoBaseHeader: true,
				ExperimentID: h.experimentID,
				SubContext:   ctx,
			},
		})
		return
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	if code.IsReferral {
		h.referral(w, r, code)
		return
	}

	h.promotion(w, r, code)
}

func (h *promoClaimHandler) referral(w http.ResponseWriter, r *http.Request, code *common.PromoCode) {
	var err error
	ctx := &refContext{}
	ctx.Ref, err = h.dataAPI.ReferralProgram(code.ID, common.PromotionTypes)
	if ctx.Ref == nil || ctx.Ref.Status == common.RSInactive {
		ctx.Message = "Sorry, the referral code is no longer active."
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	patient, err := h.dataAPI.GetPatientFromAccountID(ctx.Ref.AccountID)
	if api.IsErrNotFound(err) {
		dr, err := h.dataAPI.GetDoctorFromAccountID(ctx.Ref.AccountID)
		if api.IsErrNotFound(err) {
			www.InternalServerError(w, r, fmt.Errorf("neither doctor nor patient found for account ID %d", ctx.Ref.AccountID))
			return
		} else if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		ctx.IsDoctor = true
		ctx.ReferrerName = dr.LongDisplayName
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	} else {
		ctx.ReferrerName = patient.FirstName
	}

	www.TemplateResponse(w, http.StatusOK, h.refTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       "Referral | Spruce",
		SubContext: &homeContext{
			SubContext: ctx,
		},
	})
}

func (h *promoClaimHandler) promotion(w http.ResponseWriter, r *http.Request, code *common.PromoCode) {
	ctx := &promoContext{
		Email:  r.FormValue("email"),
		State:  r.FormValue("state"),
		Errors: make(map[string]string),
	}

	var err error
	ctx.Code = code.Code
	ctx.Promo, err = promotions.LookupPromoCode(ctx.Code, h.dataAPI, h.analyticsLogger)
	if err != promotions.InvalidCode && err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	if ctx.Promo == nil {
		ctx.Message = "Sorry, that promotion is no longer valid."
	} else {
		ctx.States, err = h.dataAPI.ListStates()
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		for _, s := range ctx.States {
			if s.Abbreviation == ctx.State {
				ctx.StateName = s.Name
				break
			}
		}

		if r.Method == "POST" {
			if ctx.Email == "" || !email.IsValidEmail(ctx.Email) {
				ctx.Errors["email"] = "Please enter a valid email address."
			}
			if ctx.State == "" {
				ctx.Errors["state"] = "Please select a state."
			}

			if len(ctx.Errors) == 0 {
				// Perform this operation asynchronously to avoid exposing existing accounts
				async := true
				ctx.SuccessMessage, err = promotions.AssociatePromoCode(ctx.Email, ctx.State, ctx.Code, h.dataAPI, h.authAPI, h.analyticsLogger, async)
				if err != nil {
					www.InternalServerError(w, r, err)
					return
				}
				ctx.Android = !strings.Contains(r.UserAgent(), "iPhone")
				ctx.Claimed = true
				inState, err := h.dataAPI.SpruceAvailableInState(ctx.State)
				if err != nil {
					www.InternalServerError(w, r, err)
					return
				}
				if !inState {
					p := url.Values{
						"email":     []string{ctx.Email},
						"state":     []string{ctx.State},
						"stateName": []string{ctx.StateName},
					}
					http.Redirect(w, r, "/r/"+ctx.Code+"/notify/state?"+p.Encode(), http.StatusSeeOther)
					return
				}
			}
		}
	}

	www.TemplateResponse(w, http.StatusOK, h.promoTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML("Claim a Promotion"),
		SubContext: &homeContext{
			NoBaseHeader: true,
			ExperimentID: h.experimentID,
			SubContext:   ctx,
		},
	})
}

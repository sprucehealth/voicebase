package home

import (
	"html/template"
	"net/http"

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

type promoNotifyAndroidHandler struct {
	dataAPI         api.DataAPI
	analyticsLogger analytics.Logger
	template        *template.Template
	experimentID    string
}

func newPromoNotifyAndroidHandler(dataAPI api.DataAPI, analyticsLogger analytics.Logger, templateLoader *www.TemplateLoader, experimentID string) http.Handler {
	return httputil.SupportedMethods(&promoNotifyAndroidHandler{
		dataAPI:         dataAPI,
		analyticsLogger: analyticsLogger,
		template:        templateLoader.MustLoadTemplate("promotions/notify_android.html", "promotions/base.html", nil),
		experimentID:    experimentID,
	}, []string{"GET", "POST"})
}

func (h *promoNotifyAndroidHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := &promoContext{
		Email:  r.FormValue("email"),
		State:  r.FormValue("state"),
		Errors: make(map[string]string),
	}

	var err error
	ctx.Code = mux.Vars(r)["code"]
	ctx.Promo, err = promotions.LookupPromoCode(ctx.Code, h.dataAPI, h.analyticsLogger)
	if err != promotions.InvalidCode && err != nil {
		www.InternalServerError(w, r, err)
		return
	}
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
				requestID := httputil.RequestID(r)
				if err := h.dataAPI.RecordForm(&common.NotifyMeForm{
					Email:    ctx.Email,
					State:    ctx.State,
					Platform: "Android",
				}, "promotion", requestID); err != nil {
					www.InternalServerError(w, r, err)
					return
				}
				ctx.Message = "Thanks! We’ll notify you when Spruce is available for Android."
			}
		}
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML("Notify Of Android Availability"),
		SubContext: &homeContext{
			NoBaseHeader: true,
			ExperimentID: h.experimentID,
			SubContext:   ctx,
		},
	})
}

package home

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type promoNotifyStateHandler struct {
	dataAPI         api.DataAPI
	analyticsLogger analytics.Logger
	template        *template.Template
}

func newPromoNotifyStateHandler(dataAPI api.DataAPI, analyticsLogger analytics.Logger, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&promoNotifyStateHandler{
		dataAPI:         dataAPI,
		analyticsLogger: analyticsLogger,
		template:        templateLoader.MustLoadTemplate("promotions/notify_state.html", "promotions/base.html", nil),
	}, []string{"GET", "POST"})
}

func (h *promoNotifyStateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := &promoContext{
		Email:     r.FormValue("email"),
		State:     r.FormValue("state"),
		StateName: r.FormValue("stateName"),
		Platform:  r.FormValue("platform"),
		Errors:    make(map[string]string),
	}

	if r.Method == "POST" {
		if ctx.Email == "" || !email.IsValidEmail(ctx.Email) {
			ctx.Errors["email"] = "Please enter a valid email address."
		}
		if ctx.Platform == "" {
			ctx.Errors["platform"] = "Please select a device."
		}

		if len(ctx.Errors) == 0 {
			requestID := httputil.RequestID(r)
			if err := h.dataAPI.RecordForm(&common.NotifyMeForm{
				Email:    ctx.Email,
				State:    ctx.State,
				Platform: ctx.Platform,
			}, "promotion", requestID); err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			ctx.Message = fmt.Sprintf("Thanks! We’ll notify you when Spruce is available in %s for %s.", ctx.StateName, ctx.Platform)
		}
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML("Notify Of Availability"),
		SubContext: &homeContext{
			NoBaseHeader: true,
			SubContext:   ctx,
		},
	})
}

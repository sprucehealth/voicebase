package admin

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type dashboardHandler struct {
	template *template.Template
}

func newDashboardHandler(templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&dashboardHandler{
		template: templateLoader.MustLoadTemplate("admin/dashboard.html", "base.html", nil),
	}, []string{"GET"})
}

func (h *dashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML("Dashboard"),
		SubContext:  &struct{}{},
	})
}

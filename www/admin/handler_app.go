package admin

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type appHandler struct {
	template *template.Template
}

func NewAppHandler(templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&appHandler{
		template: templateLoader.MustLoadTemplate("admin/app.html", "admin/base.html", nil),
	}, []string{"GET"})
}

func (h *appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title:      "Admin",
		SubContext: nil,
	})
}

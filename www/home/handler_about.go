package home

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type aboutHandler struct {
	router   *mux.Router
	template *template.Template
}

func newAboutHandler(router *mux.Router, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&aboutHandler{
		router:   router,
		template: templateLoader.MustLoadTemplate("home/about.html", "home/base.html", nil),
	}, []string{"GET"})
}

func (h *aboutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "About Spruce",
	})
}

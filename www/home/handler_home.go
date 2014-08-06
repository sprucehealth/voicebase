package home

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type homeHandler struct {
	router   *mux.Router
	template *template.Template
}

func newHomeHandler(router *mux.Router, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&homeHandler{
		router:   router,
		template: templateLoader.MustLoadTemplate("home/home.html", "home/base.html", nil),
	}, []string{"GET"})
}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Spruce",
	})
}

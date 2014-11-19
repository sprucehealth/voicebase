package home

import (
	"html"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type staticHandler struct {
	router   *mux.Router
	template *template.Template
	title    string
}

type homeContext struct {
	NoBaseHeader bool
	ExperimentID string
	SubContext   interface{}
}

func newStaticHandler(router *mux.Router, templateLoader *www.TemplateLoader, template, title string) http.Handler {
	return httputil.SupportedMethods(&staticHandler{
		router:   router,
		title:    title,
		template: templateLoader.MustLoadTemplate(template, "home/base.html", nil),
	}, []string{"GET"})
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML(html.EscapeString(h.title)),
		SubContext:  &homeContext{},
	})
}

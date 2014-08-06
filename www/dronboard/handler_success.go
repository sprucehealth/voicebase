package dronboard

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type successHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
}

func NewSuccessHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&successHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("dronboard/success.html", "dronboard/base.html", nil),
	}, []string{"GET"})
}

func (h *successHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title:      "Success| Doctor Registration | Spruce",
		SubContext: nil,
	})
}

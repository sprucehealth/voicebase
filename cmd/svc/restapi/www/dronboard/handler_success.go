package dronboard

import (
	"html/template"
	"net/http"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
)

type successHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
}

func newSuccessHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.SupportedMethods(&successHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("dronboard/success.html", "dronboard/base.html", nil),
	}, httputil.Get)
}

func (h *successHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title:      "Success| Doctor Registration | Spruce",
		SubContext: nil,
	})
}

package admin

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type rxGuideListHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
}

func NewRXGuideListHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&rxGuideListHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("admin/rxguide_list.html", "admin/base.html", nil),
	}, []string{"GET"})
}

func (h *rxGuideListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	drugs, err := h.dataAPI.ListDrugDetails()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Resource Guides",
		SubContext: &rxGuideListTemplateContext{
			Drugs: drugs,
		},
	})
}

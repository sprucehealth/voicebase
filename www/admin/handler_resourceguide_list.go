package admin

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type resourceGuideListHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
}

func NewResourceGuideListHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&resourceGuideListHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("admin/resourceguide_list.html", "admin/base.html", nil),
	}, []string{"GET"})
}

func (h *resourceGuideListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sections, guides, err := h.dataAPI.ListResourceGuides()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Resource Guides",
		SubContext: &resourceGuideListTemplateContext{
			Sections: sections,
			Guides:   guides,
		},
	})
}

package admin

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/treatment_plan"
	"github.com/sprucehealth/backend/www"
)

type rxGuideHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
}

func NewRXGuideHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&rxGuideHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("admin/rxguide.html", "admin/base.html", nil),
	}, []string{"GET"})
}

func (h *rxGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	details, err := h.dataAPI.DrugDetails(vars["ndc"])
	if err == api.NoRowsError {
		http.NotFound(w, r)
		return
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	b := &bytes.Buffer{}
	if err := treatment_plan.RenderRXGuide(b, details, nil, nil); err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: template.HTML(template.HTMLEscapeString(details.Name)),
		SubContext: &rxGuideTemplateContext{
			Details:     details,
			DetailsHTML: template.HTML(b.String()),
		},
	})
}

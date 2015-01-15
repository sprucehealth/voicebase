package admin

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type analyticsPresentationIframeHandler struct {
	dataAPI  api.DataAPI
	template *template.Template
}

func NewAnalyticsPresentationIframeHandler(dataAPI api.DataAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&analyticsPresentationIframeHandler{
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("admin/analytics_presentation_iframe.html", "base.html", nil),
	}, []string{"GET"})
}

func (h *analyticsPresentationIframeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	report, err := h.dataAPI.AnalyticsReport(id)
	if api.IsErrNotFound(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "",
		SubContext: &struct {
			Presentation template.HTML
		}{
			Presentation: template.HTML(report.Presentation),
		},
	})
}

package admin

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type dashboardHandler struct {
	adminAPI api.AdminAPI
	template *template.Template
}

func newDashboardHandler(adminAPI api.AdminAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&dashboardHandler{
		adminAPI: adminAPI,
		template: templateLoader.MustLoadTemplate("admin/dashboard.html", "base.html", nil),
	}, []string{"GET"})
}

func (h *dashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dashID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	dash, err := h.adminAPI.Dashboard(dashID)
	if api.IsErrNotFound(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML("Dashboard"),
		SubContext: &struct {
			Dashboard *common.AdminDashboard
		}{
			Dashboard: dash,
		},
	})
}

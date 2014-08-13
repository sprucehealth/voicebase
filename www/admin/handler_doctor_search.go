package admin

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type doctorSearchHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
}

func NewDoctorSearchHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&doctorSearchHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("admin/doctor_search.html", "admin/base.html", nil),
	}, []string{"GET"})
}

func (h *doctorSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var doctors []*common.DoctorSearchResult
	query := r.FormValue("q")

	if query != "" {
		var err error
		doctors, err = h.dataAPI.SearchDoctors(query)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Doctors",
		SubContext: &doctorSearchTemplateContext{
			Query:   query,
			Doctors: doctors,
		},
	})
}

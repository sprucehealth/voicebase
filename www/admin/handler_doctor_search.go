package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type doctorSearchHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

func NewDoctorSearchHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&doctorSearchHandler{
		router:  router,
		dataAPI: dataAPI,
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

	www.TemplateResponse(w, http.StatusOK, doctorSearchTemplate, &www.BaseTemplateContext{
		Title: "Doctors",
		SubContext: &doctorSearchTemplateContext{
			Query:   query,
			Doctors: doctors,
		},
	})
}

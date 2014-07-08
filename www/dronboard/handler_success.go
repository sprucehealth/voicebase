package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type successHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

func NewSuccessHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&successHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *successHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, successTemplate, &www.BaseTemplateContext{
		Title:      "Success| Doctor Registration | Spruce",
		SubContext: &successTemplateContext{},
	})
}

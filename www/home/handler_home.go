package home

import (
	"net/http"

	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type homeHandler struct {
	router *mux.Router
}

func NewHomeHandler(router *mux.Router) http.Handler {
	return httputil.SupportedMethods(&homeHandler{
		router: router,
	}, []string{"GET"})
}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, homeTemplate, &www.BaseTemplateContext{
		Title: "Spruce",
	})
}

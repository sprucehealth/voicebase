package home

import (
	"net/http"

	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type aboutHandler struct {
	router *mux.Router
}

func NewAboutHandler(router *mux.Router) http.Handler {
	return httputil.SupportedMethods(&aboutHandler{
		router: router,
	}, []string{"GET"})
}

func (h *aboutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, aboutTemplate, &www.BaseTemplateContext{
		Title: "About Spruce",
	})
}

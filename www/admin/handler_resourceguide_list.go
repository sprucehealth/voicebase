package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type resourceGuideListHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

func NewResourceGuideListHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&resourceGuideListHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *resourceGuideListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sections, guides, err := h.dataAPI.ListResourceGuides()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	www.TemplateResponse(w, http.StatusOK, resourceGuideListTemplate, &www.BaseTemplateContext{
		Title: "Resource Guides",
		SubContext: &resourceGuideListTemplateContext{
			Sections: sections,
			Guides:   guides,
		},
	})
}

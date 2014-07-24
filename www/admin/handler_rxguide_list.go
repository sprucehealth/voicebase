package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type rxGuideListHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

func NewRXGuideListHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&rxGuideListHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *rxGuideListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	drugs, err := h.dataAPI.ListDrugDetails()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	www.TemplateResponse(w, http.StatusOK, rxGuideListTemplate, &www.BaseTemplateContext{
		Title: "Resource Guides",
		SubContext: &rxGuideListTemplateContext{
			Drugs: drugs,
		},
	})
}

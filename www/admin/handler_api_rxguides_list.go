package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type rxGuidesListAPIHandler struct {
	dataAPI api.DataAPI
}

func NewRXGuideListAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&rxGuidesListAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *rxGuidesListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	drugs, err := h.dataAPI.ListDrugDetails()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	www.JSONResponse(w, r, http.StatusOK, drugs)
}

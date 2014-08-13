package admin

import (
	"bytes"
	"net/http"

	"github.com/sprucehealth/backend/common"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/treatment_plan"
	"github.com/sprucehealth/backend/www"
)

type rxGuidesAPIHandler struct {
	dataAPI api.DataAPI
}

func NewRXGuideAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&rxGuidesAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *rxGuidesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	details, err := h.dataAPI.DrugDetails(mux.Vars(r)["ndc"])
	if err == api.NoRowsError {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	var html string

	if r.FormValue("with_html") != "" {
		b := &bytes.Buffer{}
		if err := treatment_plan.RenderRXGuide(b, details, nil, nil); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		html = b.String()
	}

	www.JSONResponse(w, r, http.StatusOK, &struct {
		Guide *common.DrugDetails `json:"guide"`
		HTML  string              `json:"html"`
	}{
		Guide: details,
		HTML:  html,
	})
}

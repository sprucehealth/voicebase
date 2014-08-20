package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
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
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "ListRXGuides", nil)

	drugs, err := h.dataAPI.ListDrugDetails()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	www.JSONResponse(w, r, http.StatusOK, drugs)
}

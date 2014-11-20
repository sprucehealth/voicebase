package admin

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type analyticsReportsListAPIHandler struct {
	dataAPI api.DataAPI
}

func NewAnalyticsReportsListAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&analyticsReportsListAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *analyticsReportsListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

	if r.Method == "POST" {
		audit.LogAction(account.ID, "AdminAPI", "CreateAnalyticsReport", nil)

		var rep common.AnalyticsReport
		if err := json.NewDecoder(r.Body).Decode(&rep); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		id, err := h.dataAPI.CreateAnalyticsReport(account.ID, rep.Name, rep.Query, rep.Presentation)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, id)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "ListAnalyticsReports", nil)

	reports, err := h.dataAPI.ListAnalyticsReports()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	www.JSONResponse(w, r, http.StatusOK, reports)
}

package admin

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type analyticsReportsRunAPIHandler struct {
	dataAPI api.DataAPI
	db      *sql.DB
}

func NewAnalyticsReportsRunAPIHandler(dataAPI api.DataAPI, db *sql.DB) http.Handler {
	return httputil.SupportedMethods(&analyticsReportsRunAPIHandler{
		dataAPI: dataAPI,
		db:      db,
	}, []string{"POST"})
}

func (h *analyticsReportsRunAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reportID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "AnalyticsRunReport", map[string]interface{}{
		"report_id": reportID,
	})

	report, err := h.dataAPI.AnalyticsReport(reportID)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	runAnalyticsQuery(w, r, h.db, report.Query, nil)
}

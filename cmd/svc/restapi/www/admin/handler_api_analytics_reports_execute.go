package admin

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/mux"
)

type analyticsReportsRunAPIHandler struct {
	dataAPI api.DataAPI
	db      *sql.DB
}

func newAnalyticsReportsRunAPIHandler(dataAPI api.DataAPI, db *sql.DB) http.Handler {
	return httputil.SupportedMethods(&analyticsReportsRunAPIHandler{
		dataAPI: dataAPI,
		db:      db,
	}, httputil.Post)
}

func (h *analyticsReportsRunAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reportID, err := strconv.ParseInt(mux.Vars(r.Context())["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := www.MustCtxAccount(r.Context())
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

package admin

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type analyticsReportsRunAPIHandler struct {
	dataAPI api.DataAPI
	db      *sql.DB
}

func newAnalyticsReportsRunAPIHandler(dataAPI api.DataAPI, db *sql.DB) httputil.ContextHandler {
	return httputil.SupportedMethods(&analyticsReportsRunAPIHandler{
		dataAPI: dataAPI,
		db:      db,
	}, httputil.Post)
}

func (h *analyticsReportsRunAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	reportID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := www.MustCtxAccount(ctx)
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

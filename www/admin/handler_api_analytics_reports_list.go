package admin

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type analyticsReportsListAPIHandler struct {
	dataAPI api.DataAPI
}

func newAnalyticsReportsListAPIHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&analyticsReportsListAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Post)
}

func (h *analyticsReportsListAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)

	if r.Method == httputil.Post {
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

		httputil.JSONResponse(w, http.StatusOK, id)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "ListAnalyticsReports", nil)

	reports, err := h.dataAPI.ListAnalyticsReports()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, reports)
}

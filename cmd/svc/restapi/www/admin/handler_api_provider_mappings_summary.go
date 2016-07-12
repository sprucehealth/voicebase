package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/libs/httputil"
)

type providerMappingsSummaryHandler struct {
	dataAPI api.DataAPI
}

type providerMappingsSummaryResponse struct {
	Summary []*api.CareProviderStatePathwayMappingSummary `json:"summary"`
}

func newProviderMappingsSummaryHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&providerMappingsSummaryHandler{
		dataAPI: dataAPI,
	}, httputil.Get)
}

func (h *providerMappingsSummaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(r.Context())

	audit.LogAction(account.ID, "AdminAPI", "CareProviderStatePathwayMappingsSummary", nil)

	summary, err := h.dataAPI.CareProviderStatePathwayMappingSummary()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	// Prefer empty array to null in returned JSON
	if summary == nil {
		summary = []*api.CareProviderStatePathwayMappingSummary{}
	}
	httputil.JSONResponse(w, http.StatusOK, providerMappingsSummaryResponse{Summary: summary})
}

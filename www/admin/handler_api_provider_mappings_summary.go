package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type providerMappingsSummaryHandler struct {
	dataAPI api.DataAPI
}

type providerMappingsSummaryResponse struct {
	Summary []*api.CareProviderStatePathwayMappingSummary `json:"summary"`
}

func NewProviderMappingsSummaryHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&providerMappingsSummaryHandler{
		dataAPI: dataAPI,
	}, httputil.Get)
}

func (h *providerMappingsSummaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

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

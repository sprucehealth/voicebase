package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type providerMappingsHandler struct {
	dataAPI api.DataAPI
}

type providerMappingsResponse struct {
	Mappings []*api.CareProviderStatePathway `json:"mappings"`
}

func newProviderMappingsHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&providerMappingsHandler{
		dataAPI: dataAPI,
	}, httputil.Get)
}

func (h *providerMappingsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)

	audit.LogAction(account.ID, "AdminAPI", "ListCareProviderStatePathwayMappings", nil)

	query := &api.CareProviderStatePathwayMappingQuery{
		State:      r.FormValue("state"),
		PathwayTag: r.FormValue("pathway_tag"),
		Provider: api.Provider{
			Role: r.FormValue("provider_role"),
		},
	}

	if s := r.FormValue("provider_id"); s != "" {
		var err error
		query.Provider.ID, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			www.APIBadRequestError(w, r, "provider_id invalid")
			return
		}
	}

	mappings, err := h.dataAPI.CareProviderStatePathwayMappings(query)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	// Prefer empty array to null in returned JSON
	if mappings == nil {
		mappings = []*api.CareProviderStatePathway{}
	}
	httputil.JSONResponse(w, http.StatusOK, providerMappingsResponse{Mappings: mappings})
}

package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type resourceGuidesListAPIHandler struct {
	dataAPI api.DataAPI
}

func NewResourceGuidesListAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&resourceGuidesListAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *resourceGuidesListAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sectionsOnly := r.FormValue("sections_only") != ""

	sections, guides, err := h.dataAPI.ListResourceGuides()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	var guidesJS map[string][]*common.ResourceGuide
	if !sectionsOnly {
		guidesJS = make(map[string][]*common.ResourceGuide, len(guides))
		for sid, gs := range guides {
			guidesJS[strconv.FormatInt(sid, 10)] = gs
		}
	}

	www.JSONResponse(w, r, http.StatusOK, &struct {
		Sections []*common.ResourceGuideSection     `json:"sections"`
		Guides   map[string][]*common.ResourceGuide `json:"guides"`
	}{
		Sections: sections,
		Guides:   guidesJS,
	})
}

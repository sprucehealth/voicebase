package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type resourceGuidesListAPIHandler struct {
	dataAPI api.DataAPI
}

type resourceGuideList struct {
	Sections []*common.ResourceGuideSection     `json:"sections"`
	Guides   map[string][]*common.ResourceGuide `json:"guides"`
}

func newResourceGuidesListAPIHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&resourceGuidesListAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Put, httputil.Post)
}

func (h *resourceGuidesListAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		h.put(ctx, w, r)
	case "GET":
		h.get(ctx, w, r)
	case "POST":
		h.post(ctx, w, r)
	}
}

func (h *resourceGuidesListAPIHandler) get(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "ListResourceGuides", nil)

	withLayouts, _ := strconv.ParseBool(r.FormValue("with_layouts"))
	sectionsOnly, _ := strconv.ParseBool(r.FormValue("sections_only"))
	activeOnly, _ := strconv.ParseBool(r.FormValue("active_only"))

	var opt api.ResourceGuideListOption
	if withLayouts {
		opt |= api.RGWithLayouts
	}
	if activeOnly {
		opt |= api.RGActiveOnly
	}
	sections, guides, err := h.dataAPI.ListResourceGuides(opt)
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

	httputil.JSONResponse(w, http.StatusOK, &resourceGuideList{
		Sections: sections,
		Guides:   guidesJS,
	})
}

func (h *resourceGuidesListAPIHandler) put(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "ImportResourceGuides", nil)

	if err := r.ParseMultipartForm(maxMemory); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	f, _, err := r.FormFile("json")
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	defer f.Close()

	var js resourceGuideList
	if err := json.NewDecoder(f).Decode(&js); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	guides := make(map[int64][]*common.ResourceGuide)
	for sidStr, gs := range js.Guides {
		sid, err := strconv.ParseInt(sidStr, 10, 64)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		guides[sid] = gs
	}

	if err := h.dataAPI.ReplaceResourceGuides(js.Sections, guides); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, true)
}

func (h *resourceGuidesListAPIHandler) post(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "CreateResourceGuide", nil)

	guide := &common.ResourceGuide{}
	if err := json.NewDecoder(r.Body).Decode(guide); err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	id, err := h.dataAPI.CreateResourceGuide(guide)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	guide.ID = id
	httputil.JSONResponse(w, http.StatusOK, guide)
}

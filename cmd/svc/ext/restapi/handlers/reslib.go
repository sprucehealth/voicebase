package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type resourceGuideHandler struct {
	dataAPI api.DataAPI
}

type resourceGuideListHandler struct {
	dataAPI api.DataAPI
}

type Guide struct {
	ID       int64  `json:"id,string"`
	Title    string `json:"title"`
	PhotoURL string `json:"photo_url"`
}

type Section struct {
	ID     int64    `json:"id,string"`
	Title  string   `json:"title"`
	Guides []*Guide `json:"guides"`
}

type ListResponse struct {
	Sections []*Section `json:"sections"`
}

func NewResourceGuideHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&resourceGuideHandler{
			dataAPI: dataAPI,
		}), httputil.Get)
}

func NewResourceGuideListHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&resourceGuideListHandler{
			dataAPI: dataAPI,
		}), httputil.Get)
}

func (h *resourceGuideHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.FormValue("resource_id"), 10, 64)
	if err != nil {
		apiservice.WriteValidationError(ctx, "resource_id required and must be an integer", w, r)
		return
	}
	guide, err := h.dataAPI.GetResourceGuide(id)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError(ctx, "Guide not found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, errors.New("Failed to fetch resource guide: "+err.Error()), w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, guide.Layout)
}

func (h *resourceGuideListHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sections, guides, err := h.dataAPI.ListResourceGuides(api.RGActiveOnly)
	if err != nil {
		apiservice.WriteError(ctx, errors.New("Failed to fetch resources: "+err.Error()), w, r)
		return
	}
	res := ListResponse{
		Sections: make([]*Section, len(sections)),
	}
	for i, s := range sections {
		if gs := guides[s.ID]; len(gs) != 0 {
			sec := &Section{
				ID:     s.ID,
				Title:  s.Title,
				Guides: make([]*Guide, len(gs)),
			}
			for j, g := range gs {
				sec.Guides[j] = &Guide{
					ID:       g.ID,
					Title:    g.Title,
					PhotoURL: g.PhotoURL,
				}
			}
			res.Sections[i] = sec
		}
	}
	httputil.JSONResponse(w, http.StatusOK, &res)
}
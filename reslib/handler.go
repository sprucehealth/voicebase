package reslib

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type handler struct {
	dataAPI api.DataAPI
}

type listHandler struct {
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

func NewHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&handler{
			dataAPI: dataAPI,
		}), httputil.Get)
}

func NewListHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&listHandler{
			dataAPI: dataAPI,
		}), httputil.Get)
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.FormValue("resource_id"), 10, 64)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "resource_id required and must be an integer")
		return
	}
	guide, err := h.dataAPI.GetResourceGuide(id)
	if api.IsErrNotFound(err) {
		apiservice.WriteDeveloperError(w, http.StatusNotFound, "Guide not found")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to fetch resource: "+err.Error())
		return
	}
	httputil.JSONResponse(w, http.StatusOK, guide.Layout)
}

func (h *listHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sections, guides, err := h.dataAPI.ListResourceGuides(api.RGActiveOnly)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to fetch resources: "+err.Error())
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

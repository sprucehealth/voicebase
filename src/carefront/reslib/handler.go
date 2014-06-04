package reslib

import (
	"carefront/api"
	"carefront/apiservice"
	"net/http"
	"strconv"
)

type Handler struct {
	dataAPI api.DataAPI
}

type ListHandler struct {
	dataAPI api.DataAPI
}

type Guide struct {
	Id       int64  `json:"id,string"`
	Title    string `json:"title"`
	PhotoURL string `json:"photo_url"`
}

type Section struct {
	Id     int64    `json:"id,string"`
	Title  string   `json:"title"`
	Guides []*Guide `json:"guides"`
}

type ListResponse struct {
	Sections []*Section `json:"sections"`
}

func NewHandler(dataAPI api.DataAPI) *Handler {
	return &Handler{
		dataAPI: dataAPI,
	}
}

func NewListHandler(dataAPI api.DataAPI) *ListHandler {
	return &ListHandler{
		dataAPI: dataAPI,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.FormValue("resource_id"), 10, 64)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "resource_id required and must be an integer")
		return
	}
	guide, err := h.dataAPI.GetResourceGuide(id)
	if err == api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusNotFound, "Guide not found")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to fetch resource: "+err.Error())
		return
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, guide.Layout)
}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sections, guides, err := h.dataAPI.ListResourceGuides()
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to fetch resources: "+err.Error())
		return
	}
	res := ListResponse{
		Sections: make([]*Section, len(sections)),
	}
	for i, s := range sections {
		if gs := guides[s.Id]; len(gs) != 0 {
			sec := &Section{
				Id:     s.Id,
				Title:  s.Title,
				Guides: make([]*Guide, len(gs)),
			}
			for j, g := range gs {
				sec.Guides[j] = &Guide{
					Id:       g.Id,
					Title:    g.Title,
					PhotoURL: g.PhotoURL,
				}
			}
			res.Sections[i] = sec
		}
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &res)
}

func (*Handler) NonAuthenticated() bool {
	return true
}

func (*ListHandler) NonAuthenticated() bool {
	return true
}

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
	Id       int64  `json:"id"`
	Title    string `json:"title"`
	PhotoURL string `json:"photo_url"`
}

type ListResponse struct {
	Guides []*Guide
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
	guides, err := h.dataAPI.ListResourceGuides()
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to fetch resources: "+err.Error())
		return
	}
	res := ListResponse{
		Guides: make([]*Guide, len(guides)),
	}
	for i, g := range guides {
		res.Guides[i] = &Guide{Id: g.Id, Title: g.Title, PhotoURL: g.PhotoURL}
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &res)
}

package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/storage"
)

type thumbnailHandler struct {
	dataAPI        api.DataAPI
	staticBaseURL  string
	thumbnailStore storage.Store
}

type thumbnailRequest struct {
	Role   string `schema:"role"`
	RoleID int64  `schema:"role_id"`
	Size   string `schema:"size"`
}

func NewThumbnailHandler(dataAPI api.DataAPI, staticBaseURL string, thumbnailStore storage.Store) http.Handler {
	return &thumbnailHandler{
		dataAPI:        dataAPI,
		staticBaseURL:  staticBaseURL,
		thumbnailStore: thumbnailStore,
	}
}

func (h *thumbnailHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}
	return true, nil
}

func (h *thumbnailHandler) NonAuthenticated() bool {
	return true
}

func (h *thumbnailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req thumbnailRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	if req.Size != "small" && req.Size != "large" {
		http.NotFound(w, r)
		return
	}

	if req.Role != api.DOCTOR_ROLE && req.Role != api.MA_ROLE {
		http.NotFound(w, r)
		return
	}

	doctor, err := h.dataAPI.GetDoctorFromId(req.RoleID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	var storeID string
	if req.Size == "small" {
		storeID = doctor.SmallThumbnailID
	} else if req.Size == "large" {
		storeID = doctor.LargeThumbnailID
	}
	if storeID == "" {
		// No image set so show the place holder image
		http.Redirect(w, r, fmt.Sprintf("%s/img/doctor_placeholder_%s.png", h.staticBaseURL, req.Size), http.StatusSeeOther)
		return
	}
	url, err := h.thumbnailStore.GetSignedURL(storeID, time.Now().Add(time.Hour*24))
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

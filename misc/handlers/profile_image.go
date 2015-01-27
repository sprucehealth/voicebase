package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
)

type profileImageHandler struct {
	dataAPI       api.DataAPI
	staticBaseURL string
	imageStore    storage.Store
}

type profileImageRequest struct {
	Role   string `schema:"role"`
	RoleID int64  `schema:"role_id"`
	Type   string `schema:"type"`
	Width  int    `schema:"width"`
	Height int    `schema:"height"`
}

func NewProfileImageHandler(dataAPI api.DataAPI, staticBaseURL string, imageStore storage.Store) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&profileImageHandler{
				dataAPI:       dataAPI,
				staticBaseURL: staticBaseURL,
				imageStore:    imageStore,
			}), []string{"GET"})
}

func (h *profileImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req profileImageRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	if req.Type != "thumbnail" && req.Type != "hero" {
		http.NotFound(w, r)
		return
	}

	if req.Role != api.DOCTOR_ROLE && req.Role != api.MA_ROLE {
		http.NotFound(w, r)
		return
	}

	doctor, err := h.dataAPI.GetDoctorFromID(req.RoleID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var storeID string
	if req.Type == "thumbnail" {
		storeID = doctor.LargeThumbnailID
	} else if req.Type == "hero" {
		storeID = doctor.HeroImageID
	}
	if storeID == "" {
		// No image set so show the place holder image
		http.Redirect(w, r, fmt.Sprintf("%s/img/doctor_placeholder_%s.png", h.staticBaseURL, req.Type), http.StatusSeeOther)
		return
	}

	url, err := h.imageStore.GetSignedURL(storeID, time.Now().Add(time.Hour*24))
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

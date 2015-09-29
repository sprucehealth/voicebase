package handlers

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
	"golang.org/x/net/context"
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

func NewProfileImageHandler(dataAPI api.DataAPI, staticBaseURL string, imageStore storage.Store) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&profileImageHandler{
				dataAPI:       dataAPI,
				staticBaseURL: staticBaseURL,
				imageStore:    imageStore,
			}), httputil.Get)
}

func (h *profileImageHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req profileImageRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	if req.Type != "thumbnail" && req.Type != "hero" {
		http.NotFound(w, r)
		return
	}

	if req.Role != api.RoleDoctor && req.Role != api.RoleCC {
		http.NotFound(w, r)
		return
	}

	doctor, err := h.dataAPI.GetDoctorFromID(req.RoleID)
	if api.IsErrNotFound(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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

	rc, headers, err := h.imageStore.GetReader(storeID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	defer rc.Close()

	// TODO: provide a valid lastModified and check for "If-Modified-Since" earlier in this request
	w.Header().Set("Content-Type", headers.Get("Content-Type"))
	if cl := headers.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	httputil.FarFutureCacheHeaders(w.Header(), time.Time{})
	io.Copy(w, rc)
}

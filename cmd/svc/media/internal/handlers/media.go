package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/mime"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/httputil"
	lmedia "github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/media"
)

type mediaHandler struct {
	svc            service.Service
	mediaAPIDomain string
	maxMemory      int64
}

const (
	contentTypeHeader   = "Content-Type"
	contentLengthHeader = "Content-Length"
)

type mediaPOSTResponse struct {
	MediaID  string `json:"media_id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	MIMEType string `json:"mimetype"`
}

func (h *mediaHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Head:
		h.redirectToSource(ctx, w, r)
	case httputil.Get:
		h.redirectToSource(ctx, w, r)
	case httputil.Post:
		h.servePOST(ctx, w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *mediaHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.maxMemory); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, mimeType, err := parseMultiPartMedia("media", r)
	if err != nil {
		if file != nil {
			file.Close()
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	thumbFile, tType, err := parseMultiPartMedia("thumbnail", r)
	if thumbFile != nil {
		defer thumbFile.Close()
	}
	// If we're provided with a mimetype then make sure it's an image, otherwise assume it is
	if err == nil && tType != nil && tType.Type != "image" {
		http.Error(w, fmt.Sprintf("Media type %s is not valid for thumbnails", tType.String()), http.StatusBadRequest)
		return
	}

	meta, err := h.svc.PutMedia(ctx, file, mimeType, thumbFile)
	if err == service.ErrUnsupportedContentType {
		http.Error(w, err.Error()+" - "+mimeType.String(), http.StatusBadRequest)
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &mediaPOSTResponse{
		MediaID:  meta.MediaID.String(),
		MIMEType: meta.MIMEType,
		URL:      media.URL(h.mediaAPIDomain, meta.MediaID.String()),
		ThumbURL: media.ThumbnailURL(h.mediaAPIDomain, meta.MediaID.String(), 0, 0, false),
	})
}

func parseMultiPartMedia(formKey string, r *http.Request) (multipart.File, *mime.Type, error) {
	file, fHeaders, err := r.FormFile(formKey)
	if err != nil {
		return nil, nil, fmt.Errorf("Missing or invalid value for %s in parameters: %s", formKey, err)
	}
	mimeType, err := mime.ParseType(fHeaders.Header.Get(contentTypeHeader))
	if err != nil {
		return file, nil, fmt.Errorf("Unable to parse Content-Type for %s", formKey)
	}
	return file, mimeType, nil
}

func (h *mediaHandler) redirectToSource(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	mediaID, err := dal.ParseMediaID(mux.Vars(ctx)["id"])
	if err != nil {
		http.Error(w, "Cannot parse media id", http.StatusBadRequest)
		return
	}

	// For serving GET requests just redirect to the source with an expiring URL
	eURL, err := h.svc.ExpiringURL(ctx, mediaID, time.Minute*15)
	if errors.Cause(err) == dal.ErrNotFound || errors.Cause(err) == lmedia.ErrNotFound {
		http.NotFound(w, r)
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	http.Redirect(w, r, eURL, http.StatusSeeOther)
}

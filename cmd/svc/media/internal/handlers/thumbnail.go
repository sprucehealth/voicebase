package handlers

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"

	"golang.org/x/net/context"
)

type thumbnailHandler struct {
	svc service.Service
}

func (h *thumbnailHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	// TODO: Remove this HEAD/GET hack once we have consistent resize typing
	case httputil.Head:
		h.serveGET(ctx, w, r)
	case httputil.Get:
		h.serveGET(ctx, w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *thumbnailHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	mediaID, err := dal.ParseMediaID(mux.Vars(ctx)["id"])
	if err != nil {
		http.Error(w, "Cannot parse media id", http.StatusBadRequest)
		return
	}
	imageSize, err := parseImageSize(r)
	if err != nil {
		http.Error(w, "Cannot parse image size", http.StatusBadRequest)
		return
	}

	rc, meta, err := h.svc.GetThumbnailReader(ctx, mediaID, imageSize)
	if errors.Cause(err) == dal.ErrNotFound || errors.Cause(err) == media.ErrNotFound {
		http.NotFound(w, r)
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	defer rc.Close()
	copyWith(w, rc, int64(meta.Size), meta.MimeType, r)
}

func copyWith(w http.ResponseWriter, r io.Reader, contentLen int64, mimeType string, req *http.Request) {
	w.Header().Set(contentTypeHeader, mimeType)
	if contentLen > 0 {
		w.Header().Set(contentTypeHeader, strconv.FormatInt(contentLen, 64))
	}

	if req.Method != httputil.Head {
		httputil.FarFutureCacheHeaders(w.Header(), time.Time{})
		io.Copy(w, r)
	}
}

func parseImageSize(r *http.Request) (*media.ImageSize, error) {
	width, err := httputil.ParseFormInt(r, "width")
	if err != nil {
		return nil, err
	}
	height, err := httputil.ParseFormInt(r, "height")
	if err != nil {
		return nil, err
	}
	crop, err := httputil.ParseFormBool(r, "crop")
	if err != nil {
		return nil, err
	}
	allowScaleUp, err := httputil.ParseFormBool(r, "allow_scale_up")
	if err != nil {
		return nil, err
	}
	return &media.ImageSize{
		Width:        width,
		Height:       height,
		Crop:         crop,
		AllowScaleUp: allowScaleUp,
	}, nil
}

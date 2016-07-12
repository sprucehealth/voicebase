package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"context"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/mediautils"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/schema"
)

const (
	mediaPathFormatString = "m%v"
)

type mediaHandler struct {
	apiDomain       string
	mediaSvc        *media.ImageService
	statGetLatency  metrics.Histogram
	statPostLatency metrics.Histogram
}

// NewMedia returns a handler to perform media uploads and fetching
func NewMedia(
	apiDomain string,
	mediaSvc *media.ImageService, metricsRegistry metrics.Registry,
) httputil.ContextHandler {
	h := &mediaHandler{
		apiDomain:       apiDomain,
		mediaSvc:        mediaSvc,
		statGetLatency:  metrics.NewUnbiasedHistogram(),
		statPostLatency: metrics.NewUnbiasedHistogram(),
	}
	metricsRegistry.Add("latency/get", h.statGetLatency)
	metricsRegistry.Add("latency/post", h.statPostLatency)
	return httputil.SupportedMethods(h, httputil.Get, httputil.Post)
}

func (h *mediaHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		vars := mux.Vars(ctx)
		mediaID := vars["id"]
		if mediaID == "" {
			http.NotFound(w, r)
			return
		}
		rd, err := h.parseGETRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
		}
		h.serveGET(ctx, w, r, rd, mediaID)
	case httputil.Post:
		h.servePOST(ctx, w, r)
	}
}

func copyWithHeaders(w http.ResponseWriter, r io.Reader, contentLen int, mimeType string) {
	w.Header().Set("Content-Type", mimeType)
	if contentLen > 0 {
		w.Header().Set("Content-Length", strconv.Itoa(contentLen))
	}
	// Note: We are currently not attaching a Last-Modified header on responses
	httputil.FarFutureCacheHeaders(w.Header(), time.Now())
	io.Copy(w, r)
}

func (h *mediaHandler) parseGETRequest(r *http.Request) (*responses.MediaGETRequest, error) {
	rd := &responses.MediaGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, err
	}

	return rd, nil
}

func (h *mediaHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *responses.MediaGETRequest, mediaID string) {
	startTime := time.Now()
	defer func() {
		h.statGetLatency.Update(time.Since(startTime).Nanoseconds() / 1e3)
	}()

	id := fmt.Sprintf(mediaPathFormatString, mediaID)
	rc, meta, err := h.mediaSvc.GetReader(id, &media.ImageSize{Width: rd.Width, Height: rd.Height, Crop: rd.Crop, AllowScaleUp: rd.AllowScaleUp})
	if errors.Cause(err) == media.ErrNotFound {
		apiservice.WriteResourceNotFoundError(ctx, "media not found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	copyWithHeaders(w, rc, int(meta.Size), meta.MimeType)
}

func (h *mediaHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		h.statPostLatency.Update(time.Since(startTime).Nanoseconds() / 1e3)
	}()

	file, _, err := r.FormFile("media")
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Missing or invalid media in parameters: "+err.Error())
		return
	}
	defer file.Close()

	mediaID, err := idgen.NewID()
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	id := fmt.Sprintf(mediaPathFormatString, mediaID)
	meta, err := h.mediaSvc.PutReader(id, file)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	res := &responses.MediaPOSTResponse{
		MediaID:  mediaID,
		MediaURL: mediautils.URL(h.apiDomain, strconv.FormatInt(int64(mediaID), 10)),
		Width:    meta.Width,
		Height:   meta.Height,
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

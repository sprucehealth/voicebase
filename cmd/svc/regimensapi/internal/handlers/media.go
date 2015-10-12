package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // imported to register PNG decoder
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/schema"
	"golang.org/x/net/context"
)

const (
	mediaPathFormatString      = "m%v"
	mediaCachePathFormatString = mediaPathFormatString + "-%d-%d"
)

type mediaHandler struct {
	webDomain          string
	deterministicStore storage.DeterministicStore
	statCacheHit       *metrics.Counter
	statCacheMiss      *metrics.Counter
	statTotalLatency   metrics.Histogram
	statResizeLatency  metrics.Histogram
	statWriteLatency   metrics.Histogram
}

// NewMedia returns a handler to perform media uploads and fetching
func NewMedia(
	webDomain string,
	deterministicStore storage.DeterministicStore,
	metricsRegistry metrics.Registry,
) httputil.ContextHandler {
	h := &mediaHandler{
		webDomain:          webDomain,
		deterministicStore: deterministicStore,
		statCacheHit:       metrics.NewCounter(),
		statCacheMiss:      metrics.NewCounter(),
		statTotalLatency:   metrics.NewUnbiasedHistogram(),
		statResizeLatency:  metrics.NewUnbiasedHistogram(),
		statWriteLatency:   metrics.NewUnbiasedHistogram(),
	}
	metricsRegistry.Add("cache/hit", h.statCacheHit)
	metricsRegistry.Add("cache/miss", h.statCacheMiss)
	metricsRegistry.Add("latency/total", h.statTotalLatency)
	metricsRegistry.Add("latency/resize", h.statResizeLatency)
	metricsRegistry.Add("latency/write", h.statWriteLatency)
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

func copyWithHeaders(w http.ResponseWriter, r io.Reader, headers http.Header) {
	w.Header().Set("Content-Type", headers.Get("Content-Type"))
	if cl := headers.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	// Note: We are currently no attaching a Last-Modified header on responses
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
	mediaURL := fmt.Sprintf(mediaPathFormatString, mediaID)

	// Stream original image if no resizing is requested
	if rd.Width <= 0 && rd.Height <= 0 {
		rc, headers, err := h.deterministicStore.GetReader(h.deterministicStore.IDFromName(mediaURL))
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
		defer rc.Close()
		copyWithHeaders(w, rc, headers)
		h.statTotalLatency.Update(time.Since(startTime).Nanoseconds() / 1e3)
		return
	}

	// Check the cache
	cacheURL := fmt.Sprintf(mediaCachePathFormatString, mediaID, rd.Width, rd.Height)
	rc, headers, err := h.deterministicStore.GetReader(h.deterministicStore.IDFromName(cacheURL))
	if err == nil {
		defer rc.Close()
		h.statCacheHit.Inc(1)
		copyWithHeaders(w, rc, headers)
		h.statTotalLatency.Update(time.Since(startTime).Nanoseconds() / 1e3)
		return
	}
	h.statCacheMiss.Inc(1)

	// Resize the image since we didn't find it)
	rc, _, err = h.deterministicStore.GetReader(h.deterministicStore.IDFromName(mediaURL))
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	defer rc.Close()

	resizeStartTime := time.Now()
	resizedImg, err := media.ResizeImageFromReader(rc, rd.Width, rd.Height)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	h.statResizeLatency.Update(time.Since(resizeStartTime).Nanoseconds() / 1e3)

	w.Header().Set("Content-Type", "image/jpeg")
	// Note: We are currently no attaching a Last-Modified header on responses
	httputil.FarFutureCacheHeaders(w.Header(), time.Time{})

	// Note: Is this still relevant?
	// TODO: Dual stream encoding to cache and response once the s3 storage
	// implements streaming multi-writer. S3 requires specifying the content-length
	// on uploads so it's necessary to use a multi-part upload when the size is
	// unknown. However, the s3 package we're using currently doesn't allow setting
	// headers multi-part uploads.
	writeStartTime := time.Now()
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, resizedImg, &jpeg.Options{
		Quality: media.JPEGQuality,
	}); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// Save resized image to the cache
	go func() {
		if _, err := h.deterministicStore.Put(cacheURL, buf.Bytes(), "image/jpeg", nil); err != nil {
			golog.Errorf("Failed to write resize image to cache: %s", err.Error())
		}
	}()

	w.Write(buf.Bytes())
	h.statWriteLatency.Update(time.Since(writeStartTime).Nanoseconds() / 1e3)
	h.statTotalLatency.Update(time.Since(startTime).Nanoseconds() / 1e3)
}

func (h *mediaHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	file, handler, err := r.FormFile("media")
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Missing or invalid media in parameters: "+err.Error())
		return
	}
	defer file.Close()

	// Validate that the file is an image
	if _, _, err := image.DecodeConfig(file); err != nil {
		apiservice.WriteBadRequestError(ctx, errors.New("Unrecognized media format"), w, r)
		return
	}
	// Reset the reader after we have validated that this is an image
	if _, err := file.Seek(0, 0); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	size, err := common.SeekerSize(file)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	mediaID, err := idgen.NewID()
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	path := fmt.Sprintf(mediaPathFormatString, mediaID)
	contentType := handler.Header.Get("Content-Type")
	_, err = h.deterministicStore.PutReader(path, file, size, contentType, nil)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	res := &responses.MediaPOSTResponse{
		MediaID:  mediaID,
		MediaURL: fmt.Sprintf("%s/media/%d", strings.TrimRight(h.webDomain, "/"), mediaID),
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

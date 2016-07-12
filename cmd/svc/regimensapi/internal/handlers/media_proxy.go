package handlers

import (
	"net/http"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/mediaproxy"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/schema"
)

type mediaProxyHandler struct {
	mpSvc          *mediaproxy.Service
	statGetLatency metrics.Histogram
}

// NewMediaProxy returns a handler to perform 3rd media proxying
func NewMediaProxy(
	mpSvc *mediaproxy.Service,
	metricsRegistry metrics.Registry,
) http.Handler {
	h := &mediaProxyHandler{
		mpSvc:          mpSvc,
		statGetLatency: metrics.NewUnbiasedHistogram(),
	}
	metricsRegistry.Add("latency/get", h.statGetLatency)
	return httputil.SupportedMethods(h, httputil.Get)
}

func (h *mediaProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		h.statGetLatency.Update(time.Since(startTime).Nanoseconds() / 1e3)
	}()

	mediaID := mux.Vars(r.Context())["id"]

	var rd responses.MediaGETRequest
	if err := r.ParseForm(); err != nil {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}
	if err := schema.NewDecoder().Decode(&rd, r.Form); err != nil {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}

	sz := &media.ImageSize{Width: rd.Width, Height: rd.Height, Crop: rd.Crop, AllowScaleUp: rd.AllowScaleUp}
	rc, meta, err := h.mpSvc.ImageReader(mediaID, sz)
	if errors.Cause(err) == media.ErrNotFound {
		apiservice.WriteResourceNotFoundError("Media not found", w, r)
		return
	} else if _, ok := errors.Cause(err).(mediaproxy.ErrFetchFailed); ok {
		golog.Infof("Media proxy fetch failed: %s", err)
		apiservice.WriteResourceNotFoundError("Fetch failed", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	copyWithHeaders(w, rc, meta.Size, meta.MimeType)
}

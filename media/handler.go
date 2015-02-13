package media

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/bamiaux/rez"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
)

const jpegQuality = 95

var resizeFilter = rez.NewLanczosFilter(3)

type SubImage interface {
	SubImage(r image.Rectangle) image.Image
}

type handler struct {
	dataAPI            api.DataAPI
	store              *Store
	cacheStore         storage.DeterministicStore
	expirationDuration time.Duration
	statCacheHit       *metrics.Counter
	statCacheMiss      *metrics.Counter
	statTotalLatency   metrics.Histogram
	statResizeLatency  metrics.Histogram
	statReadLatency    metrics.Histogram
	statWriteLatency   metrics.Histogram
}

type uploadResponse struct {
	MediaID  int64  `json:"media_id,string"`
	PhotoID  int64  `json:"photo_id,string"`
	MediaURL string `json:"media_url"`
	PhotoURL string `json:"photo_url"`
}

type mediaRequest struct {
	MediaID    int64  `schema:"media_id,required"`
	Signature  string `schema:"sig,required"`
	ExpireTime int64  `schema:"expires,required"`
	Width      int    `schema:"width"`
	Height     int    `schema:"height"`
}

type mediaResponse struct {
	MediaType string `json:"media_type"`
	MediaURL  string `json:"media_url"`
}

func NewHandler(
	dataAPI api.DataAPI,
	store *Store,
	cacheStore storage.DeterministicStore,
	expirationDuration time.Duration,
	metricsRegistry metrics.Registry,
) http.Handler {
	h := &handler{
		dataAPI:            dataAPI,
		store:              store,
		cacheStore:         cacheStore,
		expirationDuration: expirationDuration,
		statCacheHit:       metrics.NewCounter(),
		statCacheMiss:      metrics.NewCounter(),
		statTotalLatency:   metrics.NewUnbiasedHistogram(),
		statResizeLatency:  metrics.NewUnbiasedHistogram(),
		statReadLatency:    metrics.NewUnbiasedHistogram(),
		statWriteLatency:   metrics.NewUnbiasedHistogram(),
	}
	metricsRegistry.Add("cache/hit", h.statCacheHit)
	metricsRegistry.Add("cache/miss", h.statCacheMiss)
	metricsRegistry.Add("latency/total", h.statTotalLatency)
	metricsRegistry.Add("latency/resize", h.statResizeLatency)
	metricsRegistry.Add("latency/read", h.statReadLatency)
	metricsRegistry.Add("latency/write", h.statWriteLatency)
	return httputil.SupportedMethods(apiservice.AuthorizationRequired(h), []string{"GET", "POST"})
}

func (h *handler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	switch r.Method {
	case "GET":
		req := &mediaRequest{}
		if err := apiservice.DecodeRequestData(req, r); err != nil {
			return false, apiservice.NewValidationError(err.Error())
		}
		if req.ExpireTime < time.Now().UTC().Unix() {
			return false, apiservice.NewAccessForbiddenError()
		}
		if !h.store.ValidateSignature(req.MediaID, req.ExpireTime, req.Signature) {
			return false, apiservice.NewAccessForbiddenError()
		}
		ctxt.RequestCache[apiservice.RequestData] = req
	case "POST":
		role := ctxt.Role
		var personID int64
		switch role {
		case api.DOCTOR_ROLE, api.MA_ROLE:
			doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
			if err != nil {
				return false, err
			}
			personID, err = h.dataAPI.GetPersonIDByRole(role, doctorID)
			if err != nil {
				return false, err
			}
		case api.PATIENT_ROLE:
			patientID, err := h.dataAPI.GetPatientIDFromAccountID(ctxt.AccountID)
			if err != nil {
				return false, err
			}
			personID, err = h.dataAPI.GetPersonIDByRole(api.PATIENT_ROLE, patientID)
			if err != nil {
				return false, err
			}
		default:
			return false, apiservice.NewAccessForbiddenError()
		}
		ctxt.RequestCache[apiservice.PersonID] = personID
	}
	return true, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
	case "POST":
		h.post(w, r)
	}
}

func copyWithHeaders(w http.ResponseWriter, r io.Reader, headers http.Header, lastModified time.Time) {
	w.Header().Set("Content-Type", headers.Get("Content-Type"))
	if cl := headers.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	httputil.FarFutureCacheHeaders(w.Header(), lastModified)
	io.Copy(w, r)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)
	req := ctx.RequestCache[apiservice.RequestData].(*mediaRequest)

	startTime := time.Now()

	media, err := h.dataAPI.GetMedia(req.MediaID)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError("Media not found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// TODO: Check "If-Modified-Since"

	// Stream original image if no resizing is requested
	if req.Width <= 0 && req.Height <= 0 {
		rc, headers, err := h.store.GetReader(media.URL)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		defer rc.Close()
		copyWithHeaders(w, rc, headers, media.Uploaded)
		h.statTotalLatency.Update(time.Since(startTime).Nanoseconds() / 1e3)
		return
	}

	// Resizing is request so first check the cache

	cacheKey := fmt.Sprintf("%d-%dx%d", req.MediaID, req.Width, req.Height)

	if h.cacheStore != nil {
		rc, headers, err := h.cacheStore.GetReader(h.cacheStore.IDFromName(cacheKey))
		if err == nil {
			defer rc.Close()
			h.statCacheHit.Inc(1)
			copyWithHeaders(w, rc, headers, media.Uploaded)
			h.statTotalLatency.Update(time.Since(startTime).Nanoseconds() / 1e3)
			return
		}
	}
	h.statCacheMiss.Inc(1)

	// Not in the cache so generate the requested size

	readStartTime := time.Now()
	rc, _, err := h.store.GetReader(media.URL)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	defer rc.Close()
	img, _, err := image.Decode(rc)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	h.statReadLatency.Update(time.Since(readStartTime).Nanoseconds() / 1e3)

	/*
		The resize calculation/algorithm works like this:

		If we're only given one dimension (width or height) then calculate the other one
		based on the aspect ratio of the original image. In this case no cropping is performed
		and all we need to do is resize the image to the calculated size.

		If both width and height are provided then we'll likely need to crop unless aspect
		ratio of the request width and height matches the original image exactly. If cropping
		is requires then the original image is first resized to be large enough (but no larger)
		in order to fit the request image size, and it's then cropped. For instance a 640x480
		original image being request to resize to 320x320 is first resized to 426x320 and then
		cropped from the center to the final size of 320x320.
	*/

	width := req.Width
	height := req.Height

	// Never return a larger image than the original.
	if width > img.Bounds().Dx() {
		width = img.Bounds().Dx()
	}
	if height > img.Bounds().Dy() {
		height = img.Bounds().Dy()
	}

	// If only given one dimension then calculate the other dimension based on the aspect ratio.
	var crop bool
	if width <= 0 {
		width = img.Bounds().Dx() * height / img.Bounds().Dy()
	} else if height <= 0 {
		height = img.Bounds().Dy() * width / img.Bounds().Dx()
	} else {
		crop = true
	}

	resizeWidth := width
	resizeHeight := height
	if crop {
		imgRatio := float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())
		cropRatio := float64(width) / float64(height)
		if imgRatio == cropRatio {
			crop = false
		} else if imgRatio > cropRatio {
			resizeWidth = img.Bounds().Dx() * height / img.Bounds().Dy()
		} else {
			resizeHeight = img.Bounds().Dy() * width / img.Bounds().Dx()
		}
	}

	// Create a new image that matches the format of the original. The rez
	// package can only resize into the same format as the source.
	var resizedImg image.Image
	rr := image.Rect(0, 0, resizeWidth, resizeHeight)
	switch m := img.(type) {
	case *image.YCbCr:
		resizedImg = image.NewYCbCr(rr, m.SubsampleRatio)
	case *image.RGBA:
		resizedImg = image.NewRGBA(rr)
	case *image.NRGBA:
		resizedImg = image.NewNRGBA(rr)
	case *image.Gray:
		resizedImg = image.NewGray(rr)
	default:
		// Shouldn't ever have other types (and pretty much always YCbCr) since
		// the media is (at least at the moment) all captured from a camera and
		// encoded as JPEG.
		apiservice.WriteError(fmt.Errorf("image type %T not supported", img), w, r)
		return
	}

	resizeStartTime := time.Now()
	if err := rez.Convert(resizedImg, img, resizeFilter); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	h.statResizeLatency.Update(time.Since(resizeStartTime).Nanoseconds() / 1e3)

	if crop {
		// It's safe to assume that resizeImg implements the SubImage interface
		// because above we matched on specific image types that all have the
		// SubImage method.
		x0 := (resizeWidth - width) / 2
		y0 := (resizeHeight - height) / 2
		resizedImg = resizedImg.(SubImage).SubImage(image.Rect(x0, y0, x0+width, y0+height))
	}

	w.Header().Set("Content-Type", "image/jpeg")
	httputil.FarFutureCacheHeaders(w.Header(), media.Uploaded)

	// TODO: Dual stream encoding to cache and response once the s3 storage
	// implements streaming multi-writer. S3 requires specifying the content-length
	// on uploads so it's necessary to use a multi-part upload when the size is
	// unknown. However, the s3 package we're using currently doesn't allow setting
	// headers multi-part uploads.

	writeStartTime := time.Now()
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, resizedImg, &jpeg.Options{
		Quality: jpegQuality,
	}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Save resized image to the cache
	go func() {
		headers := map[string][]string{
			"Content-Type": []string{"image/jpeg"},
		}
		if _, err := h.cacheStore.Put(cacheKey, buf.Bytes(), headers); err != nil {
			golog.Errorf("Failed to write resize image to cache: %s", err.Error())
		}
	}()

	w.Write(buf.Bytes())
	h.statWriteLatency.Update(time.Since(writeStartTime).Nanoseconds() / 1e3)

	h.statTotalLatency.Update(time.Since(startTime).Nanoseconds() / 1e3)
}

func (h *handler) post(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	personID := ctxt.RequestCache[apiservice.PersonID].(int64)
	file, handler, err := r.FormFile("media")
	if err != nil {
		file, handler, err = r.FormFile("photo")
		if err != nil {
			apiservice.WriteUserError(w, http.StatusBadRequest, "Missing or invalid media in parameters: "+err.Error())
			return
		}
	}
	defer file.Close()

	size, err := common.SeekerSize(file)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	uid := make([]byte, 16)
	if _, err := rand.Read(uid); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	name := "media-" + hex.EncodeToString(uid)
	contentType := handler.Header.Get("Content-Type")
	headers := http.Header{
		"Content-Type": []string{contentType},
	}

	url, err := h.store.PutReader(name, file, size, headers)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	id, err := h.dataAPI.AddMedia(personID, url, contentType)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	signedURL, err := h.store.SignedURL(id, h.expirationDuration)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	res := &uploadResponse{
		MediaID:  id,
		PhotoID:  id,
		MediaURL: signedURL,
		PhotoURL: signedURL,
	}
	apiservice.WriteJSON(w, res)
}

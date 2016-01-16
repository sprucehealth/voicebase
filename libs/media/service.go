// Package media implements a service to store and resize images.
package media

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // imported to register decoder
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/imageutil"
	"github.com/sprucehealth/backend/libs/storage"
)

// ErrNotFound is returned when the requested media was not found
var ErrNotFound = errors.New("media: media not found")

// ErrInvalidImage is returned when an image cannot be decoded or resized
type ErrInvalidImage struct {
	Err error
}

func (e ErrInvalidImage) Error() string {
	return fmt.Sprintf("media: invalid image: %s", e.Err)
}

const (
	widthHeader         = "x-amz-meta-width"
	heightHeader        = "x-amz-meta-height"
	mimeTypeHeader      = "Content-Type"
	contentLengthHeader = "Content-Length"
)

// Size is a requested size for an image.
type Size struct {
	Width        int
	Height       int
	AllowScaleUp bool
	Crop         bool
}

// Meta is is media metadata
type Meta struct {
	MimeType string
	Width    int
	Height   int
	Size     int // in bytes of the encoded image
}

// Service implements a media storage service.
type Service struct {
	store               storage.DeterministicStore
	storeCache          storage.DeterministicStore
	maxWidth, maxHeight int
}

// New returns a new initialized media service.
func New(store, storeCache storage.DeterministicStore, maxWidth, maxHeight int) *Service {
	return &Service{
		store:      store,
		storeCache: storeCache,
		maxWidth:   maxWidth,
		maxHeight:  maxHeight,
	}
}

// Put stores an image.
func (s *Service) Put(id string, img image.Image) (*Meta, error) {
	// If the image is larger than allowed then resize and store
	if s.isTooLarge(img.Bounds().Dx(), img.Bounds().Dy()) {
		var err error
		img, err = imageutil.ResizeImage(img, s.maxWidth, s.maxHeight, &imageutil.Options{AllowScaleUp: false, Crop: false})
		if err != nil {
			return nil, errors.Trace(ErrInvalidImage{Err: err})
		}
	}
	return s.storeOriginal(id, img)
}

// PutReader stores an image and returns the image metadata It's often better than
// Put as it can avoid re-encoding the image when not necessary.
func (s *Service) PutReader(id string, r io.ReadSeeker) (*Meta, error) {
	cnf, imf, _, err := imageutil.DecodeImageConfigAndExif(r)
	if err != nil {
		return nil, errors.Trace(ErrInvalidImage{Err: err})
	}
	if _, err := r.Seek(0, 0); err != nil {
		return nil, errors.Trace(err)
	}

	// If the image is larger than allowed then resize and store
	if s.isTooLarge(cnf.Width, cnf.Height) {
		var err error
		img, _, err := imageutil.ResizeImageFromReader(r, s.maxWidth, s.maxHeight, &imageutil.Options{AllowScaleUp: false, Crop: false})
		if err != nil {
			return nil, errors.Trace(ErrInvalidImage{Err: err})
		}
		return s.storeOriginal(id, img)
	}

	// Figure out the size of the data
	size, err := r.Seek(0, os.SEEK_END)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if _, err := r.Seek(0, os.SEEK_SET); err != nil {
		return nil, errors.Trace(err)
	}

	meta := &Meta{
		MimeType: "image/" + imf, // This works for all stdlib decoders but might fail for others. Probably fine though.
		Width:    cnf.Width,
		Height:   cnf.Height,
		Size:     int(size),
	}
	_, err = s.store.PutReader(id, r, size, meta.MimeType, map[string]string{
		widthHeader:  strconv.Itoa(cnf.Width),
		heightHeader: strconv.Itoa(cnf.Height),
	})
	return meta, errors.Trace(err)
}

func (s *Service) storeOriginal(id string, img image.Image) (*Meta, error) {
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: imageutil.JPEGQuality}); err != nil {
		return nil, errors.Trace(err)
	}
	if _, err := s.store.Put(id, buf.Bytes(), "image/jpeg", imgHeaders(img)); err != nil {
		return nil, errors.Trace(err)
	}
	return &Meta{
		MimeType: "image/jpeg",
		Width:    img.Bounds().Dx(),
		Height:   img.Bounds().Dy(),
		Size:     buf.Len(),
	}, nil
}

// Get returns the decoded and optionally sized image and related metadata.
func (s *Service) Get(id string, size *Size) (image.Image, *Meta, error) {
	rc, meta, err := s.GetReader(id, size)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	img, _, err := imageutil.DecodeAndOrient(rc)
	return img, meta, errors.Trace(err)
}

// GetReader returns a reader for the requested size of image and the mimetype.
// If size is nil then the original image is returned.
func (s *Service) GetReader(id string, size *Size) (io.ReadCloser, *Meta, error) {
	// If requested original then our job is easy.
	if size == nil || (size.Width <= 0 && size.Height <= 0) {
		rc, header, err := s.store.GetReader(s.store.IDFromName(id))
		if errors.Cause(err) == storage.ErrNoObject {
			return nil, nil, errors.Trace(ErrNotFound)
		} else if err != nil {
			return nil, nil, errors.Trace(err)
		}
		return rc, metaFromHeaders(header), nil
	}

	// Check for size class in the store cache
	sizeID := sizeID(id, size)
	rc, header, err := s.storeCache.GetReader(s.storeCache.IDFromName(sizeID))
	if err == nil {
		return rc, metaFromHeaders(header), nil
	}
	if err != storage.ErrNoObject {
		golog.Errorf("media: failed to fetch size '%s': %s", sizeID, err)
	}

	// Fetch the original since we didn't have the requested size already stored
	rc, header, err = s.store.GetReader(s.store.IDFromName(id))
	if errors.Cause(err) == storage.ErrNoObject {
		return nil, nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, nil, errors.Trace(err)
	}
	defer rc.Close()
	img, _, err := imageutil.ResizeImageFromReader(rc, size.Width, size.Height, &imageutil.Options{AllowScaleUp: size.AllowScaleUp, Crop: size.Crop})
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	meta := &Meta{
		MimeType: header.Get(mimeTypeHeader),
		Width:    img.Bounds().Dx(),
		Height:   img.Bounds().Dy(),
	}

	// Encode the sized image
	buf := &bytes.Buffer{}
	switch meta.MimeType {
	case "image/png":
		if err := png.Encode(buf, img); err != nil {
			return nil, nil, errors.Trace(err)
		}
	default: // Use JPEG as the default format
		meta.MimeType = "image/jpeg"
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: imageutil.JPEGQuality}); err != nil {
			return nil, nil, errors.Trace(err)
		}
	}
	meta.Size = buf.Len()

	// Cache the sized image in the background
	bytes := buf.Bytes()
	conc.Go(func() {
		if _, err := s.storeCache.Put(sizeID, bytes, meta.MimeType, imgHeaders(img)); err != nil {
			golog.Errorf("media: failed to store size '%s': %s", sizeID, err)
		}
	})

	return ioutil.NopCloser(buf), meta, nil
}

func (s *Service) isTooLarge(width, height int) bool {
	return (s.maxWidth > 0 && width > s.maxWidth) || (s.maxHeight > 0 && height > s.maxHeight)
}

func sizeID(id string, size *Size) string {
	return fmt.Sprintf("%s-%dx%d-up_%t-crop_%t", id, size.Width, size.Height, size.AllowScaleUp, size.Crop)
}

func imgHeaders(img image.Image) map[string]string {
	return map[string]string{
		widthHeader:  strconv.Itoa(img.Bounds().Dx()),
		heightHeader: strconv.Itoa(img.Bounds().Dy()),
	}
}

func metaFromHeaders(h http.Header) *Meta {
	width, _ := strconv.Atoi(h.Get(widthHeader))
	height, _ := strconv.Atoi(h.Get(heightHeader))
	size, _ := strconv.Atoi(h.Get(contentLengthHeader))
	return &Meta{
		MimeType: h.Get(mimeTypeHeader),
		Width:    width,
		Height:   height,
		Size:     size,
	}
}

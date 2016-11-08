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
	"strconv"

	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/imageutil"
	"github.com/sprucehealth/backend/libs/storage"
)

// ErrInvalidImage is returned when an image cannot be decoded or resized
type ErrInvalidImage struct {
	Err error
}

func (e ErrInvalidImage) Error() string {
	return fmt.Sprintf("media: invalid image: %s", e.Err)
}

// ImageSize is a requested size for an image.
type ImageSize struct {
	Width        int
	Height       int
	AllowScaleUp bool
	Crop         bool
}

// ImageMeta is is media metadata
type ImageMeta struct {
	Name     string
	MimeType string
	Width    int
	Height   int
	Size     uint64 // in bytes of the encoded image
	URL      string
}

// ImageService implements a media storage service.
type ImageService struct {
	store               storage.DeterministicStore
	storeCache          storage.DeterministicStore
	maxWidth, maxHeight int
}

// NewImageService returns a new initialized media service.
func NewImageService(store, storeCache storage.DeterministicStore, maxWidth, maxHeight int) *ImageService {
	return &ImageService{
		store:      store,
		storeCache: storeCache,
		maxWidth:   maxWidth,
		maxHeight:  maxHeight,
	}
}

// Put stores an image.
func (s *ImageService) Put(id string, img image.Image) (*ImageMeta, error) {
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
func (s *ImageService) PutReader(id string, r io.ReadSeeker) (*ImageMeta, error) {
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
	size, err := SeekerSize(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	mimeType := "image/" + imf
	url, err := s.store.PutReader(id, r, size, mimeType, map[string]string{
		widthHeader:  strconv.Itoa(cnf.Width),
		heightHeader: strconv.Itoa(cnf.Height),
	})
	meta := &ImageMeta{
		MimeType: mimeType, // This works for all stdlib decoders but might fail for others. Probably fine though.
		Width:    cnf.Width,
		Height:   cnf.Height,
		Size:     uint64(size),
		URL:      url,
	}
	return meta, errors.Trace(err)
}

// Copy a stored image
func (s *ImageService) Copy(dstID, srcID string) (string, error) {
	if err := s.store.Copy(s.store.IDFromName(dstID), s.store.IDFromName(srcID)); err != nil {
		if errors.Cause(err) == storage.ErrNoObject {
			return "", ErrNotFound
		}
		return "", errors.Trace(err)
	}
	return s.store.IDFromName(dstID), nil
}

func (s *ImageService) storeOriginal(id string, img image.Image) (*ImageMeta, error) {
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: imageutil.JPEGQuality}); err != nil {
		return nil, errors.Trace(err)
	}
	url, err := s.store.Put(id, buf.Bytes(), "image/jpeg", imgHeaders(img))
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &ImageMeta{
		MimeType: "image/jpeg",
		Width:    img.Bounds().Dx(),
		Height:   img.Bounds().Dy(),
		Size:     uint64(buf.Len()),
		URL:      url,
	}, nil
}

// Get returns the decoded and optionally sized image and related metadata.
func (s *ImageService) Get(id string, size *ImageSize) (image.Image, *ImageMeta, error) {
	rc, meta, err := s.GetReader(id, size)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	img, _, err := imageutil.DecodeAndOrient(rc)
	return img, meta, errors.Trace(err)
}

// GetMeta returns the metadata associated with a media entry
func (s *ImageService) GetMeta(id string) (*ImageMeta, error) {
	h, err := s.store.GetHeader(s.store.IDFromName(id))
	if err != nil {
		return nil, err
	}
	return metaFromHeaders(h), nil
}

// GetReader returns a reader for the requested size of image and the mimetype.
// If size is nil then the original image is returned.
func (s *ImageService) GetReader(id string, size *ImageSize) (io.ReadCloser, *ImageMeta, error) {
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
	meta := &ImageMeta{
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
	meta.Size = uint64(buf.Len())

	// Cache the sized image in the background
	bytes := buf.Bytes()
	conc.Go(func() {
		if _, err := s.storeCache.Put(sizeID, bytes, meta.MimeType, imgHeaders(img)); err != nil {
			golog.Errorf("media: failed to store size '%s': %s", sizeID, err)
		}
	})

	return ioutil.NopCloser(buf), meta, nil
}

// URL returns the URL from the underlying deterministic storage system
func (s *ImageService) URL(id string) string {
	return s.store.IDFromName(id)
}

func (s *ImageService) isTooLarge(width, height int) bool {
	return (s.maxWidth > 0 && width > s.maxWidth) || (s.maxHeight > 0 && height > s.maxHeight)
}

func sizeID(id string, size *ImageSize) string {
	return fmt.Sprintf("%s-%dx%d-up_%t-crop_%t", id, size.Width, size.Height, size.AllowScaleUp, size.Crop)
}

func imgHeaders(img image.Image) map[string]string {
	return map[string]string{
		widthHeader:  strconv.Itoa(img.Bounds().Dx()),
		heightHeader: strconv.Itoa(img.Bounds().Dy()),
	}
}

func metaFromHeaders(h http.Header) *ImageMeta {
	width, _ := strconv.Atoi(h.Get(widthHeader))
	height, _ := strconv.Atoi(h.Get(heightHeader))
	size, _ := strconv.Atoi(h.Get(contentLengthHeader))
	return &ImageMeta{
		Name:     h.Get(originalNameHeader),
		MimeType: h.Get(mimeTypeHeader),
		Width:    width,
		Height:   height,
		Size:     uint64(size),
	}
}

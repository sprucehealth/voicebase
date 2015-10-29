package media

import (
	"image"
	"testing"

	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/test"
)

func init() {
	conc.Testing = true
}

func TestService(t *testing.T) {
	store := storage.NewTestStore(nil)
	storeCache := storage.NewTestStore(nil)
	svc := New(store, storeCache, 128, 128)

	// Store image smaller than max size
	var img image.Image
	img = image.NewRGBA(image.Rect(0, 0, 80, 64))
	meta, err := svc.Put("1", img)
	test.OK(t, err)
	test.Equals(t, "image/jpeg", meta.MimeType)
	test.Equals(t, 80, meta.Width)
	test.Equals(t, 64, meta.Height)
	_, _, err = store.Get("1")
	test.OK(t, err)

	// Store image larger than max size
	img = image.NewRGBA(image.Rect(0, 0, 256, 128))
	meta, err = svc.Put("2", img)
	test.OK(t, err)
	test.Equals(t, "image/jpeg", meta.MimeType)
	test.Equals(t, 128, meta.Width)
	test.Equals(t, 64, meta.Height)
	_, _, err = store.Get("2")
	test.OK(t, err)

	img, meta, err = svc.Get("1", nil)
	test.OK(t, err)
	test.Equals(t, "image/jpeg", meta.MimeType)
	test.Equals(t, 80, meta.Width)
	test.Equals(t, 64, meta.Height)
	test.Assert(t, meta.Size > 0, "Size should be > 0")
	test.Equals(t, meta.Width, img.Bounds().Dx())
	test.Equals(t, meta.Height, img.Bounds().Dy())

	img, meta, err = svc.Get("2", nil)
	test.OK(t, err)
	test.Equals(t, "image/jpeg", meta.MimeType)
	test.Equals(t, 128, meta.Width)
	test.Equals(t, 64, meta.Height)
	test.Assert(t, meta.Size > 0, "Size should be > 0")
	test.Equals(t, meta.Width, img.Bounds().Dx())
	test.Equals(t, meta.Height, img.Bounds().Dy())

	// Cropped
	sc := &Size{Width: 32, Height: 32, Crop: true}
	img, meta, err = svc.Get("1", sc)
	test.OK(t, err)
	test.Equals(t, "image/jpeg", meta.MimeType)
	test.Equals(t, 32, meta.Width)
	test.Equals(t, 32, meta.Height)
	test.Assert(t, meta.Size > 0, "Size should be > 0")
	test.Equals(t, meta.Width, img.Bounds().Dx())
	test.Equals(t, meta.Height, img.Bounds().Dy())
	rc, _, err := storeCache.GetReader(sizeID("1", sc))
	test.OK(t, err)
	img, imf, err := image.Decode(rc)
	test.OK(t, err)
	test.Equals(t, meta.Width, img.Bounds().Dx())
	test.Equals(t, meta.Height, img.Bounds().Dy())
	test.Equals(t, "jpeg", imf)

	// Cached version
	sc = &Size{Width: 32, Height: 32, Crop: true}
	img, meta, err = svc.Get("1", sc)
	test.OK(t, err)
	test.Equals(t, "image/jpeg", meta.MimeType)
	test.Equals(t, 32, meta.Width)
	test.Equals(t, 32, meta.Height)
	test.Assert(t, meta.Size > 0, "Size should be > 0")
	test.Equals(t, meta.Width, img.Bounds().Dx())
	test.Equals(t, meta.Height, img.Bounds().Dy())

	// Bounded
	sc = &Size{Width: 32, Height: 32, Crop: false}
	img, meta, err = svc.Get("2", sc)
	test.OK(t, err)
	test.Equals(t, "image/jpeg", meta.MimeType)
	test.Equals(t, 32, meta.Width)
	test.Equals(t, 16, meta.Height)
	test.Assert(t, meta.Size > 0, "Size should be > 0")
	test.Equals(t, meta.Width, img.Bounds().Dx())
	test.Equals(t, meta.Height, img.Bounds().Dy())
	_, _, err = storeCache.GetReader(sizeID("2", sc))
	test.OK(t, err)
}

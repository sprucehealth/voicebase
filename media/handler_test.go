package media

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
)

var testJPEG []byte

func init() {
	img := image.NewYCbCr(image.Rect(0, 0, 640, 480), image.YCbCrSubsampleRatio422)
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		panic(err)
	}
	testJPEG = buf.Bytes()
}

type dataAPI struct {
	api.DataAPI
	media map[int64]*common.Media
}

func (d *dataAPI) GetMedia(mediaID int64) (*common.Media, error) {
	m := d.media[mediaID]
	if m == nil {
		return nil, api.ErrNotFound("media")
	}
	return m, nil
}

func TestHandlerGet(t *testing.T) {
	store := storage.NewTestStore(
		map[string]*storage.TestObject{
			"image-123": {
				Data: testJPEG,
			},
		},
	)
	signer, err := sig.NewSigner([][]byte{[]byte("xxx")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mediaStore := NewStore("http://example.com", signer, store)

	dapi := &dataAPI{
		media: map[int64]*common.Media{
			123: &common.Media{
				ID:  123,
				URL: "image-123",
			},
		},
	}

	h := NewHandler(dapi, mediaStore, store, time.Hour, metrics.NewRegistry())

	// Missing arguments

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status %d for empty request, got %d", http.StatusBadRequest, w.Code)
	}

	// Bad signature

	r, err = http.NewRequest("GET", "/?media_id=123&expires=99999999999&sig=eHh4", nil)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected status %d for bad signature, got %d", http.StatusForbidden, w.Code)
	}

	// Valid but expired

	sig, err := signer.Sign(makeSignedMsg(123, 1234))
	if err != nil {
		t.Fatal(err)
	}
	params := url.Values{
		"media_id": []string{"123"},
		"expires":  []string{"1234"},
		"sig":      []string{base64.URLEncoding.EncodeToString(sig)},
	}
	r, err = http.NewRequest("GET", "/?"+params.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected status %d for bad signature, got %d", http.StatusForbidden, w.Code)
	}

	// Valid - not resized

	ur, err := mediaStore.SignedURL(123, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r, err = http.NewRequest("GET", ur, nil)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d for valid request, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Equal(w.Body.Bytes(), testJPEG) {
		t.Fatal("Body does not match")
	}

	// Valid - resized with crop

	r, err = http.NewRequest("GET", ur+"&width=320&height=320", nil)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d for valid request, got %d", http.StatusOK, w.Code)
	}
	if img, _, err := image.Decode(w.Body); err != nil {
		t.Fatal(err)
	} else if img.Bounds().Dx() != 320 || img.Bounds().Dy() != 320 {
		t.Fatalf("Expected width,height of %d,%d got %d,%d", 320, 320, img.Bounds().Dx(), img.Bounds().Dy())
	}

	// Valid - resized fixed width

	r, err = http.NewRequest("GET", ur+"&width=320", nil)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d for valid request, got %d", http.StatusOK, w.Code)
	}
	if img, _, err := image.Decode(w.Body); err != nil {
		t.Fatal(err)
	} else if img.Bounds().Dx() != 320 || img.Bounds().Dy() != 240 {
		t.Fatalf("Expected width,height of %d,%d got %d,%d", 320, 240, img.Bounds().Dx(), img.Bounds().Dy())
	}

	// Valid - resized fixed height

	r, err = http.NewRequest("GET", ur+"&height=320", nil)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d for valid request, got %d", http.StatusOK, w.Code)
	}
	if img, _, err := image.Decode(w.Body); err != nil {
		t.Fatal(err)
	} else if img.Bounds().Dx() != 426 || img.Bounds().Dy() != 320 {
		t.Fatalf("Expected width,height of %d,%d got %d,%d", 426, 320, img.Bounds().Dx(), img.Bounds().Dy())
	}
}

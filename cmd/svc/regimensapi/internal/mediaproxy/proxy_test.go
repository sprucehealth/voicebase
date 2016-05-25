package mediaproxy

import (
	"bytes"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/test"
)

type roundTripper struct {
	urlToRes func(url string) *http.Response
	urlToErr map[string]error
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	earl := req.URL.String()
	if e := rt.urlToErr[earl]; e != nil {
		return nil, e
	}
	if r := rt.urlToRes(earl); r != nil {
		return r, nil
	}
	return &http.Response{
		Body:       ioutil.NopCloser(bytes.NewReader(nil)),
		StatusCode: http.StatusNotFound,
	}, nil
}

func TestService(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 128, 64))
	imgBuf := &bytes.Buffer{}
	test.OK(t, png.Encode(imgBuf, img))

	liveURL := "http://sprucehealth.com/img-live.jpg"
	deadURL := "http://sprucehealth.com/img-dead.jpg"
	errURL := "http://sprucehealth.com/img-err.jpg"

	store := storage.NewTestStore(nil)
	msvc := media.NewImageService(store, store, 0, 0)
	dal := NewMemoryDAL()
	rt := &roundTripper{
		urlToRes: func(earl string) *http.Response {
			switch earl {
			case liveURL:
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader(imgBuf.Bytes())),
				}
			}
			return nil
		},
		urlToErr: map[string]error{
			errURL: errors.New("Buffalo buffalo Buffalo buffalo buffalo buffalo Buffalo buffalo"),
		},
	}
	svc := New(msvc, dal, &http.Client{Transport: rt})

	// Should filter out bad URLs and keep the good
	med, err := svc.LookupByURL([]string{liveURL, "badurl", "http://regimems.com/fewfewfwefwfw.png"})
	test.OK(t, err)
	test.Equals(t, 2, len(med))
	m := med[liveURL]
	test.Equals(t, "_q-DEX7MXsZYYDz63Ycwtw", m.ID)
	test.Equals(t, liveURL, m.URL)
	test.Equals(t, StatusNotFetched, m.Status)

	// Check known URLs
	med, err = svc.LookupByURL([]string{liveURL, deadURL, errURL})
	test.OK(t, err)
	test.Equals(t, 3, len(med))
	m = med[liveURL]
	test.Equals(t, "_q-DEX7MXsZYYDz63Ycwtw", m.ID)
	test.Equals(t, liveURL, m.URL)
	test.Equals(t, StatusNotFetched, m.Status)
	m = med[deadURL]
	test.Equals(t, "IjE18UlMhDg3D6fnHRy_dw", m.ID)
	test.Equals(t, deadURL, m.URL)
	test.Equals(t, StatusNotFetched, m.Status)

	liveID := med[liveURL].ID
	deadID := med[deadURL].ID
	errID := med[errURL].ID

	// Non-existant ID
	_, _, err = svc.ImageReader("abc", nil)
	test.Equals(t, media.ErrNotFound, errors.Cause(err))

	// Fetch fail (404)
	_, _, err = svc.ImageReader(deadID, nil)
	ef, ok := errors.Cause(err).(ErrFetchFailed)
	test.Assert(t, ok, "Error should be ErrFetchFailed not %T: %s", err, err)
	test.Equals(t, true, ef.Permanent)
	test.Equals(t, "Not Found", ef.Reason)

	// Fetch fail (error during GET)
	_, _, err = svc.ImageReader(errID, nil)
	ef, ok = errors.Cause(err).(ErrFetchFailed)
	test.Assert(t, ok, "Error should be ErrFetchFailed not %T: %s", err, err)
	test.Equals(t, false, ef.Permanent)
	test.Equals(t, "request failed: Get http://sprucehealth.com/img-err.jpg: Buffalo buffalo Buffalo buffalo buffalo buffalo Buffalo buffalo", ef.Reason)

	// Fetch success
	rc, m, err := svc.ImageReader(liveID, nil)
	test.Equals(t, nil, err)
	test.Equals(t, liveID, m.ID)
	test.Equals(t, liveURL, m.URL)
	test.Equals(t, StatusStored, m.Status)
	test.Equals(t, 128, m.Width)
	test.Equals(t, 64, m.Height)
	test.Equals(t, imgBuf.Len(), m.Size)
	test.Equals(t, "image/png", m.MimeType)
	b, err := ioutil.ReadAll(rc)
	test.OK(t, err)
	test.Equals(t, b, imgBuf.Bytes())
}

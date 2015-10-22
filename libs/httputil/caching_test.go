package httputil

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestETag(t *testing.T) {
	tag := GenETag("abc")
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	test.Equals(t, false, CheckAndSetETag(w, r, tag))
	test.Equals(t, strconv.Quote(tag), w.Header().Get("ETag"))
	r.Header.Set("If-None-Match", w.Header().Get("ETag"))
	test.Equals(t, true, CheckAndSetETag(w, r, tag))
}

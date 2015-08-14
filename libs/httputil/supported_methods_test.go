package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestSupportedMethods(t *testing.T) {
	h := SupportedMethods(ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {}), Get, Put)
	ctx := context.Background()

	r, err := http.NewRequest(Get, "/", nil)
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)

	r, err = http.NewRequest(Post, "/", nil)
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusMethodNotAllowed, w)
	test.Equals(t, "GET, PUT", w.Header().Get("Allow"))

	r, err = http.NewRequest(Options, "/", nil)
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, "GET, PUT", w.Header().Get("Allow"))
}

package apiservice

import (
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestQueryableMux(t *testing.T) {
	mux := NewQueryableMux()
	mux.Handle("/pattern", http.NotFoundHandler())
	mux.Handle("/pattern2", http.NotFoundHandler())
	test.Assert(t, mux.IsSupportedPath("/pattern"), "expected path /pattern to be supported")
	test.Assert(t, mux.IsSupportedPath("/pattern2"), "expected path /pattern2 to be supported")
	test.Assert(t, mux.IsSupportedPath("/pattern3") == false, "expected path /pattern3 to not be supported")
	test.Equals(t, len([]string{"/pattern", "/pattern2"}), len(mux.SupportedPaths()))
}

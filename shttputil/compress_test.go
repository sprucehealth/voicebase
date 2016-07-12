package shttputil

import (
	"context"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/test"
)

// TODO: Figure out how to compatmentalize this within the test fn
type testCompressResponseExpected string

const (
	testCompressResponseStarting     testCompressResponseExpected = "STARTING"
	testCompressResponseCompressed   testCompressResponseExpected = "COMPRESSED"
	testCompressResponseUncompressed testCompressResponseExpected = "UNCOMPRESSED"
)

var testCompressResponseResult = testCompressResponseStarting

type uncompressedHandler struct{}

func (uc *uncompressedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	testCompressResponseResult = testCompressResponseUncompressed
}

type compressedHandler struct{}

func (uc *compressedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	testCompressResponseResult = testCompressResponseCompressed
}

func cWrapper(http.Handler) http.Handler {
	return &compressedHandler{}
}

func TestCompressResponse(t *testing.T) {
	uc := &uncompressedHandler{}
	cases := map[string]struct {
		Context  context.Context
		Expected testCompressResponseExpected
	}{
		"iOSCompressedLowestVersion": {
			Context: devicectx.WithSpruceHeaders(context.Background(), &device.SpruceHeaders{
				Platform:   device.IOS,
				AppVersion: &encoding.Version{},
			}),
			Expected: testCompressResponseCompressed,
		},
		"AndroidUncompressedVersion": {
			Context: devicectx.WithSpruceHeaders(context.Background(), &device.SpruceHeaders{
				Platform:   device.Android,
				AppVersion: &encoding.Version{Major: 1, Minor: 1},
			}),
			Expected: testCompressResponseUncompressed,
		},
		"AndroidCompressedVersion": {
			Context: devicectx.WithSpruceHeaders(context.Background(), &device.SpruceHeaders{
				Platform:   device.Android,
				AppVersion: &encoding.Version{Major: 1, Minor: 2},
			}),
			Expected: testCompressResponseCompressed,
		},
	}

	r, err := http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	for cn, c := range cases {
		CompressResponse(uc, cWrapper).ServeHTTP(nil, r.WithContext(c.Context))
		test.EqualsCase(t, cn, c.Expected, testCompressResponseResult)
		testCompressResponseResult = testCompressResponseStarting
	}
}

package shttputil

import (
	"net/http"
	"testing"

	"context"

	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
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

func (uc *uncompressedHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	testCompressResponseResult = testCompressResponseUncompressed
}

type compressedHandler struct{}

func (uc *compressedHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	testCompressResponseResult = testCompressResponseCompressed
}

func cWrapper(httputil.ContextHandler) httputil.ContextHandler {
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

	for cn, c := range cases {
		CompressResponse(uc, cWrapper).ServeHTTP(c.Context, nil, nil)
		test.EqualsCase(t, cn, c.Expected, testCompressResponseResult)
		testCompressResponseResult = testCompressResponseStarting
	}
}

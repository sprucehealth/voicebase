package shttputil

import (
	"net/http"

	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

// CompressResponse adds compression to an http response for clients currently supporting it in the Spruce app collection
func CompressResponse(h httputil.ContextHandler, cWrapper func(httputil.ContextHandler) httputil.ContextHandler) httputil.ContextHandler {
	ch := cWrapper(h)
	return httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		sh := devicectx.SpruceHeaders(ctx)
		// TODO: don't compress response for Android app < 1.2. Remove this when no longer needed.
		if sh.Platform != device.Android || sh.AppVersion.GreaterThanOrEqualTo(&encoding.Version{Major: 1, Minor: 2, Patch: 0}) {
			ch.ServeHTTP(ctx, w, r)
		} else {
			h.ServeHTTP(ctx, w, r)
		}
	})
}

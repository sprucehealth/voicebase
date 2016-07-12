package shttputil

import (
	"net/http"

	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/encoding"
)

// CompressResponse adds compression to an http response for clients currently supporting it in the Spruce app collection
func CompressResponse(h http.Handler, cWrapper func(http.Handler) http.Handler) http.Handler {
	ch := cWrapper(h)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sh := devicectx.SpruceHeaders(r.Context())
		// TODO: don't compress response for Android app < 1.2. Remove this when no longer needed.
		if sh.Platform != device.Android || sh.AppVersion.GreaterThanOrEqualTo(&encoding.Version{Major: 1, Minor: 2, Patch: 0}) {
			ch.ServeHTTP(w, r)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

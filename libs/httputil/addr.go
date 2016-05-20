package httputil

import (
	"net/http"
	"strings"
)

// UnknownRemoteAddr is returned when the remote address isn't present in the request or headers
const UnknownRemoteAddr = "UNKNOWN"

// RemoteAddrFromRequest returns the remote address for the request and attempts to discover this if behind a proxy
func RemoteAddrFromRequest(r *http.Request, behindProxy bool) string {
	var ra string
	if behindProxy {
		ra = r.Header.Get("X-Forwarded-For")
		if ix := strings.IndexByte(ra, ','); ix > 0 {
			ra = ra[:ix]
		}
		ra = strings.TrimSpace(ra)
		if ra == "" {
			return UnknownRemoteAddr
		}
	} else if r.RemoteAddr != "" {
		ra = r.RemoteAddr
	} else {
		return UnknownRemoteAddr
	}
	// Remove port from address if included (TODO: this is likely wrong for IPv6)
	if idx := strings.LastIndexByte(ra, ':'); idx > 0 {
		ra = ra[:idx]
	}
	return ra
}

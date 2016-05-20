package httputil

import (
	"net/http"
	"strings"
)

// UnknownRemoteAddr is returned when the remote address isn't present in the request or headers
const UnknownRemoteAddr = "UNKNOWN"

// RemoteAddrFromRequest returns the remote address for the request and attempts to discover this if behind a proxy
func RemoteAddrFromRequest(r *http.Request, behindProxy bool) string {
	if behindProxy {
		addrs := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		if len(addrs) < 1 {
			return UnknownRemoteAddr
		}
		return addrs[0]
	}
	if r.RemoteAddr == "" {
		return UnknownRemoteAddr
	}
	return r.RemoteAddr
}

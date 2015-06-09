package httputil

import (
	"net/http"
	"strconv"
	"time"
)

const (
	futureMaxAge          = "max-age=315360000" // 10*365*24*60*60
	futureExpiresDuration = time.Hour * 24 * 364
)

// FarFutureCacheHeaders sets the Expires and Cache-Control (max-age) headers to
// a year and 10 years respectively. The Last-Modified header is also set
// if the provided time is not zero.
func FarFutureCacheHeaders(h http.Header, lastModified time.Time) {
	if !lastModified.IsZero() {
		h.Set("Last-Modified", lastModified.Format(time.RFC1123))
	}
	h.Set("Expires", time.Now().UTC().Add(futureExpiresDuration).Format(time.RFC1123))
	h.Set("Cache-Control", futureMaxAge)
}

// CacheHeaders sets the Expires and Cache-Control (max-age) headers to the
// provided expiration date. The Last-Modified header is also set
// if the provided time is not zero.
func CacheHeaders(h http.Header, lastModified time.Time, expires time.Duration) {
	if !lastModified.IsZero() {
		h.Set("Last-Modified", lastModified.Format(time.RFC1123))
	}
	h.Set("Expires", time.Now().UTC().Add(expires).Format(time.RFC1123))
	h.Set("Cache-Control", "max-age="+strconv.FormatInt(int64(expires.Seconds()), 10))
}

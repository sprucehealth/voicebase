package httputil

import (
	"crypto/md5"
	"encoding/base64"
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

// NoCache sets max age to 0 and appends to no-cache attribute
func NoCache(h http.Header) {
	h.Set("Cache-Control", "max-age=0, no-cache")
}

// CheckAndSetETag compares the etag in the request with one generated from the provided
// tag. If the tags match then true is returned otherwise CheckAndSetETag returns false.
// It also writes the provided tag to the response ETag header.
func CheckAndSetETag(w http.ResponseWriter, r *http.Request, tag string) bool {
	w.Header().Set("ETag", strconv.Quote(tag))
	reqTagStr, err := strconv.Unquote(r.Header.Get("If-None-Match"))
	if err != nil {
		return false
	}
	return reqTagStr == tag
}

// GenETag generates an etag from the provided string. It does so by user a
// collision-resistant hash and converting to Base64
func GenETag(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

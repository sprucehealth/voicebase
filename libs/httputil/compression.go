package httputil

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type decompressRequestHandler struct {
	h http.Handler
}

// DecompressRequest wraps a handler to take care of decompressing
// requests when Content-Ending is gzip.
func DecompressRequest(h http.Handler) http.Handler {
	return &decompressRequestHandler{h: h}
}

func (ch *decompressRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Header.Get("Content-Encoding") == "gzip" {
		r.Body = &gzipReadCloser{rc: r.Body}
		defer r.Body.Close() // Only closes the gzip reader. The http server handles closing the real Body.
	}
	ch.h.ServeHTTP(w, r)
}

type compressResponseHandler struct {
	h http.Handler
}

func CompressResponse(h http.Handler) http.Handler {
	return &compressResponseHandler{h: h}
}

func (ch *compressResponseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		ch.h.ServeHTTP(w, r)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")
	rw := &gzipResponseWriter{ResponseWriter: w}
	defer rw.Close() // Only closes the gzip writer. The http server handles closing the real ResponseWriter.

	ch.h.ServeHTTP(rw, r)
}

type gzipReadCloser struct {
	rc io.ReadCloser
	zr io.ReadCloser
}

func (gz *gzipReadCloser) Read(b []byte) (int, error) {
	if gz.zr == nil {
		var err error
		gz.zr, err = gzip.NewReader(gz.rc)
		if err != nil {
			return 0, err
		}
	}
	return gz.zr.Read(b)
}

func (gz *gzipReadCloser) Close() error {
	return gz.zr.Close()
}

type gzipResponseWriter struct {
	http.ResponseWriter
	zw io.WriteCloser
}

func (gz *gzipResponseWriter) Write(b []byte) (int, error) {
	if gz.zw == nil {
		gz.zw = gzip.NewWriter(gz.ResponseWriter)
	}

	h := gz.ResponseWriter.Header()
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", http.DetectContentType(b))
	}

	return gz.zw.Write(b)
}

func (gz *gzipResponseWriter) Close() error {
	return gz.zw.Close()
}

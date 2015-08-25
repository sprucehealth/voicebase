package httputil

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/context"
)

var (
	gzipReaderPool sync.Pool
	gzipWriterPool sync.Pool
)

// compressedResponseTypes lists the mimetypes for resposnes that should be compressed.
// All text/* mimetypes are compressed by default and should not be included in this list.
var compressedResponseTypes = []string{
	"application/json", "application/javascript", "application/xml",
	"application/atom+xml", "application/rss+xml",
}

type decompressRequestHandler struct {
	h ContextHandler
}

// DecompressRequest wraps a handler to take care of decompressing
// requests when Content-Ending is gzip.
func DecompressRequest(h ContextHandler) ContextHandler {
	return &decompressRequestHandler{h: h}
}

func (ch *decompressRequestHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Header.Get("Content-Encoding") == "gzip" {
		r.Body = &gzipReadCloser{rc: r.Body}
		defer r.Body.Close() // Only closes the gzip reader. The http server handles closing the real Body.
	}
	ch.h.ServeHTTP(ctx, w, r)
}

type compressResponseHandler struct {
	h ContextHandler
}

// CompressResponse wraps a handler to take care of compressing
// responses when the content-type is of a compressible type (e.g. json, html)
func CompressResponse(h ContextHandler) ContextHandler {
	return &compressResponseHandler{h: h}
}

func (ch *compressResponseHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		ch.h.ServeHTTP(ctx, w, r)
		return
	}

	rw := &gzipResponseWriter{ResponseWriter: w}
	defer rw.Close() // Only closes the gzip writer. The http server handles closing the real ResponseWriter.

	ch.h.ServeHTTP(ctx, rw, r)
}

type gzipReadCloser struct {
	rc io.ReadCloser
	zr *gzip.Reader
}

func (gz *gzipReadCloser) Read(b []byte) (int, error) {
	if gz.zr == nil {
		var zr *gzip.Reader
		if r := gzipReaderPool.Get(); r != nil {
			zr = r.(*gzip.Reader)
			if err := zr.Reset(gz.rc); err != nil {
				return 0, err
			}
		} else {
			var err error
			zr, err = gzip.NewReader(gz.rc)
			if err != nil {
				return 0, err
			}
		}
		gz.zr = zr
	}
	return gz.zr.Read(b)
}

func (gz *gzipReadCloser) Close() error {
	if gz.zr != nil {
		err := gz.zr.Close()
		gzipReaderPool.Put(gz.zr)
		gz.zr = nil
		return err
	}
	return nil
}

type gzipResponseWriter struct {
	http.ResponseWriter
	zw            *gzip.Writer
	wroteHeader   bool
	notCompressed bool
}

func (gz *gzipResponseWriter) Write(b []byte) (int, error) {
	if !gz.wroteHeader {
		h := gz.ResponseWriter.Header()
		if h.Get("Content-Type") == "" {
			h.Set("Content-Type", http.DetectContentType(b))
		}
		gz.WriteHeader(http.StatusOK)
	}

	if gz.notCompressed {
		return gz.ResponseWriter.Write(b)
	}

	if gz.zw == nil {
		if zw := gzipWriterPool.Get(); zw != nil {
			gz.zw = zw.(*gzip.Writer)
			gz.zw.Reset(gz.ResponseWriter)
		} else {
			gz.zw = gzip.NewWriter(gz.ResponseWriter)
		}
	}

	return gz.zw.Write(b)
}

func (gz *gzipResponseWriter) Close() error {
	if gz.zw != nil {
		err := gz.zw.Close()
		gz.zw.Reset(nil)
		gzipWriterPool.Put(gz.zw)
		gz.zw = nil
		return err
	}
	return nil
}

func (gz *gzipResponseWriter) WriteHeader(status int) {
	gz.wroteHeader = true

	if compressContentType(gz.ResponseWriter.Header().Get("Content-Type")) {
		h := gz.ResponseWriter.Header()
		h.Del("Content-Length") // Remove any set content-length since it'll be inaccurate
		h.Set("Content-Encoding", "gzip")
		h.Set("Vary", "Accept-Encoding")
	} else {
		gz.notCompressed = true
	}

	gz.ResponseWriter.WriteHeader(status)
}

func compressContentType(contentType string) bool {
	if idx := strings.IndexByte(contentType, ';'); idx >= 0 {
		contentType = contentType[:idx]
	}
	if contentType == "" {
		return false
	}
	if strings.HasPrefix(contentType, "text/") {
		return true
	}
	for _, ct := range compressedResponseTypes {
		if contentType == ct {
			return true
		}
	}
	return false
}

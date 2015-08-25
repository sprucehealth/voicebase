package httputil

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"
)

func TestDecompressRequest(t *testing.T) {
	h := DecompressRequest(ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(w, r.Body); err != nil {
			t.Fatal(err)
		}
	}))

	req, err := http.NewRequest("POST", "/", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), rec, req)
	if body := rec.Body.String(); body != "hello" {
		t.Errorf("Expected echo of '%s'. Got '%s'", "hello", body)
	}

	b := &bytes.Buffer{}
	w := gzip.NewWriter(b)
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req, err = http.NewRequest("POST", "/", b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Encoding", "gzip")
	rec = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), rec, req)
	if body := rec.Body.String(); body != "hello" {
		t.Errorf("Expected echo of compressed '%s'. Got '%s'", "hello", body)
	}
}

func TestCompressResponse(t *testing.T) {
	// Compressable mimetype

	responseContentType := "text/plain"
	h := CompressResponse(ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", responseContentType)
		w.Write([]byte("hello"))
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), rec, req)
	if body := rec.Body.String(); body != "hello" {
		t.Errorf("Expected uncompressed body of '%s'. Got '%s'", "hello", body)
	}

	rec = httptest.NewRecorder()
	req.Header.Set("Accept-Encoding", "gzip,deflate")
	h.ServeHTTP(context.Background(), rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != responseContentType {
		t.Errorf("Expected content-type of '%s'. Got '%s'", responseContentType, ct)
	}
	if ce := rec.Header().Get("Content-Encoding"); ce != "gzip" {
		t.Errorf("Expected content-encoding of 'gzip'. Got '%s'", ce)
	}
	if rec.Body.String() == "hello" {
		t.Errorf("Expected compressed body")
	} else {
		d, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		b, err := ioutil.ReadAll(d)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != "hello" {
			t.Fatalf("Failed to decompress response")
		}
	}

	// Uncompressable mimetype

	responseContentType = "not/compressed"
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip,deflate")
	rec = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != responseContentType {
		t.Errorf("Expected content-type of '%s'. Got '%s'", responseContentType, ct)
	}
	if ce := rec.Header().Get("Content-Encoding"); ce != "" {
		t.Errorf("Expected content-encoding of ''. Got '%s'", ce)
	}
	if rec.Body.String() != "hello" {
		t.Errorf("Expected uncompressed body")
	}
}

func TestUncompressableResponse(t *testing.T) {
	h := CompressResponse(ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("hello"))
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip,deflate")
	rec := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), rec, req)
	if body := rec.Body.String(); body != "hello" {
		t.Errorf("Expected uncompressed body of '%s'. Got '%s'", "hello", body)
	}
}

type benchResponseWriter struct {
	headers http.Header
}

func (w *benchResponseWriter) Header() http.Header       { return w.headers }
func (*benchResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (*benchResponseWriter) WriteHeader(int)             {}

func BenchmarkCompressResponse(b *testing.B) {
	res := make([]byte, 256)
	for i := range res {
		res[i] = 'a'
	}
	resContentType := []string{"text/plain"}
	h := CompressResponse(ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header()["Content-Type"] = resContentType
		w.Write(res)
	}))
	ctx := context.Background()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		b.Fatal(err)
	}
	r.Header.Set("Accept-Encoding", "gzip,deflate")
	w := &benchResponseWriter{headers: http.Header{}}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(ctx, w, r)
	}
}

func BenchmarkDecompressRequest(b *testing.B) {
	h := DecompressRequest(ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		io.Copy(ioutil.Discard, r.Body)
	}))
	ctx := context.Background()
	rawBody := make([]byte, 256)
	for i := range rawBody {
		rawBody[i] = 'a'
	}
	buf := &bytes.Buffer{}
	gzw := gzip.NewWriter(buf)
	if _, err := gzw.Write(rawBody); err != nil {
		b.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		b.Fatal(err)
	}
	body := bytes.NewReader(buf.Bytes())
	r, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		b.Fatal(err)
	}
	r.ContentLength = int64(buf.Len())
	r.Header.Set("Content-Encoding", "gzip")
	w := nullResponseWriter{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		body.Seek(0, 0)
		r.Body = ioutil.NopCloser(body)
		h.ServeHTTP(ctx, w, r)
	}
}

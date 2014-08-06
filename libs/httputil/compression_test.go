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
)

func TestDecompressRequest(t *testing.T) {
	h := DecompressRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(w, r.Body); err != nil {
			t.Fatal(err)
		}
	}))

	req, err := http.NewRequest("POST", "/", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
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
	h.ServeHTTP(rec, req)
	if body := rec.Body.String(); body != "hello" {
		t.Errorf("Expected echo of compressed '%s'. Got '%s'", "hello", body)
	}
}

func TestCompressResponse(t *testing.T) {
	h := CompressResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if body := rec.Body.String(); body != "hello" {
		t.Errorf("Expected uncompressed body of '%s'. Got '%s'", "hello", body)
	}

	rec = httptest.NewRecorder()
	req.Header.Set("Accept-Encoding", "gzip,deflate")
	h.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("Expected content-type of 'text/plain; charset=utf-8'. Got '%s'", ct)
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
}

func TestUncompressableResponse(t *testing.T) {
	h := CompressResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("hello"))
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip,deflate")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if body := rec.Body.String(); body != "hello" {
		t.Errorf("Expected uncompressed body of '%s'. Got '%s'", "hello", body)
	}
}

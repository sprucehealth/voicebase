package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
)

func TestLoggingHandler(t *testing.T) {
	var lastEvent RequestEvent
	var events int
	h := RequestIDHandler(LoggingHandler(ContextHandlerFunc(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/panic" {
				panic("OH NOES")
			}
			w.WriteHeader(http.StatusNotImplemented)
		}),
		func(ctx context.Context, ev *RequestEvent) {
			lastEvent = *ev
			events++
		}))
	ctx := context.Background()
	r, err := http.NewRequest("GET", "/patho?a=b", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	if w.Code != http.StatusNotImplemented {
		t.Fatal("Expected StatusNotImplemented")
	}
	if events != 1 {
		t.Fatal("No event logged")
	}
	if lastEvent.StatusCode != w.Code {
		t.Fatalf("Expected status %d got %d", w.Code, lastEvent.StatusCode)
	}
	if lastEvent.Request.Method != r.Method {
		t.Fatalf("Expected method %s got %s", r.Method, lastEvent.Request.Method)
	}
	if lastEvent.URL.String() != r.URL.String() {
		t.Fatalf("Expected url %s got %s", r.URL.String(), lastEvent.URL.String())
	}

	// Panic

	r, err = http.NewRequest("HEAD", "/panic?a=b", nil)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatal("Expected StatusInternalServerError")
	}
	if events != 2 {
		t.Fatal("No event logged")
	}
	if lastEvent.StatusCode != w.Code {
		t.Fatalf("Expected status %d got %d", w.Code, lastEvent.StatusCode)
	}
	if lastEvent.Request.Method != r.Method {
		t.Fatalf("Expected method %s got %s", r.Method, lastEvent.Request.Method)
	}
	if lastEvent.URL.String() != r.URL.String() {
		t.Fatalf("Expected url %s got %s", r.URL.String(), lastEvent.URL.String())
	}
	if lastEvent.Panic != "OH NOES" {
		t.Fatalf("Expected 'OH NOES' got '%s'", lastEvent.Panic)
	}
	if len(lastEvent.StackTrace) == 0 {
		t.Fatal("Stack trace missing")
	}
}

type nullResponseWriter struct{}

func (nullResponseWriter) Header() http.Header         { return http.Header{} }
func (nullResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (nullResponseWriter) WriteHeader(int)             {}

func BenchmarkLoggingHandler(b *testing.B) {
	h := LoggingHandler(ContextHandlerFunc(
		func(context.Context, http.ResponseWriter, *http.Request) {}),
		func(context.Context, *RequestEvent) {})
	ctx := context.Background()
	r, err := http.NewRequest("GET", "/patho?a=b", nil)
	if err != nil {
		b.Fatal(err)
	}
	w := nullResponseWriter{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(ctx, w, r)
	}
}

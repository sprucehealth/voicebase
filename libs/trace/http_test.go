package trace

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"context"
)

func TestHTTP(t *testing.T) {
	var ht *Trace
	h := HTTPHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ht, _ = FromContext(ctx)
	})
	hc := HTTPContext(h, false, "family")

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	hc.ServeHTTP(w, r)
	if ht == nil {
		t.Fatal("Handler failed to set trace")
	}
}

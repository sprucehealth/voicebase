package home

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/analytics"
)

func TestAnalyticsHandler(t *testing.T) {
	al := analytics.DebugLogger{T: t}
	reg := metrics.NewRegistry()
	h := newAnalyticsHandler(al, reg)

	r, err := http.NewRequest("GET", "/?event=abc&role=admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if !bytes.Equal(w.Body.Bytes(), logoImage) {
		t.Error("Body did not match logo image")
	}

	reg.Do(func(name string, metric interface{}) error {
		switch name {
		case "events/received":
			if n := metric.(*metrics.Counter).Count(); n != 1 {
				t.Errorf("Expected 1 received event got %d", n)
			}
		case "events/dropped":
			if n := metric.(*metrics.Counter).Count(); n != 0 {
				t.Errorf("Expected 0 dropped events got %d", n)
			}
		default:
			t.Fatalf("Unexpected stat %s", name)
		}
		return nil
	})
}

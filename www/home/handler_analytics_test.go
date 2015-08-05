package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/test"
)

func TestAnalyticsHandler(t *testing.T) {
	al := analytics.DebugLogger{T: t}
	reg := metrics.NewRegistry()
	h := newAnalyticsHandler(al, reg)

	r, err := http.NewRequest("GET", "/?event=abc&foo=bar", nil)
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)

	test.Assert(t, bytes.Equal(w.Body.Bytes(), logoImage), "Body did not match logo image")

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

	body, err := json.Marshal(&analyticsAPIRequest{
		CurrentTime: float64(time.Now().Unix()),
		Events: []event{
			{Name: "xyz", Properties: properties{"foo": "bar"}},
		},
	})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)

	reg.Do(func(name string, metric interface{}) error {
		switch name {
		case "events/received":
			if n := metric.(*metrics.Counter).Count(); n != 2 {
				t.Errorf("Expected 1 received event got %d", n-1)
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

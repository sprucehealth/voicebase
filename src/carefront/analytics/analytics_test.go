package analytics

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

type testLogger struct {
	events map[string][]interface{}
}

func (l *testLogger) WriteEvents(category string, events []interface{}) {
	if l.events == nil {
		l.events = make(map[string][]interface{})
	}
	l.events[category] = append(l.events[category], events...)
}

func (l *testLogger) Start() error {
	return nil
}

func (l *testLogger) Stop() error {
	return nil
}

func (l *testLogger) clear() {
	l.events = nil
}

func TestHandler(t *testing.T) {
	lg := &testLogger{}
	h, err := NewHandler(lg, metrics.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBuffer([]byte(fmt.Sprintf(`
		[
			{
				"event": "click",
				"properties": {
					"time": %d,
					"session_id": "123abc",
					"extra": "foo"
				}
			}
		]
	`, time.Now().Unix())))
	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("S-Version", "Patient;Feature;0.9.0;000105")
	req.Header.Set("S-OS", "iOS;7.1.1")
	req.Header.Set("S-Device", "Phone;iPhone6,1;640;1136;2.0")
	req.Header.Set("S-Device-ID", "12345678-1234-1234-1234-123456789abc")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatalf("Expected 200 got %d", res.Code)
	}
	if n := h.statEventsReceived.Count(); n != 1 {
		t.Fatalf("Expected to receive 1 event. Got %d", n)
	}
	if n := h.statEventsDropped.Count(); n != 0 {
		t.Fatalf("Expected to drop 0 events. Got %d", n)
	}
	if n := len(lg.events["client"]); n != 1 {
		t.Fatalf("Expected 1 event to be recorded. Got %d", n)
	}
	t.Logf("%+v", lg.events["client"][0])
}

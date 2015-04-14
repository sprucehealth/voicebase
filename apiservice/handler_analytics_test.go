package apiservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/analytics"
)

type testPublisher struct {
	events map[string][]analytics.Event
}

func (p *testPublisher) Publish(el interface{}) error {
	er, ok := el.(analytics.Eventer)
	if !ok {
		fmt.Println("Fail")
		return fmt.Errorf("Couldn't cast contents as []analytics.Event")
	}
	e := er.Events()
	if p.events == nil {
		p.events = make(map[string][]analytics.Event)
	}
	for _, ee := range e {
		p.events[ee.Category()] = append(p.events[ee.Category()], ee)
	}
	return nil
}

func (p *testPublisher) PublishAsync(el interface{}) {
	p.Publish(el)
}

func TestHandler(t *testing.T) {
	pub := &testPublisher{}
	h := newAnalyticsHandler(pub, metrics.NewRegistry())
	now := float64(time.Now().UnixNano()) / 1e9
	body := bytes.NewBuffer([]byte(fmt.Sprintf(`
		{
			"current_time": %f,
			"events": [
				{
					"event": "click",
					"properties": {
						"time": %f,
						"session_id": "123abc",
						"extra": "foo"
					}
				}
			]
		}
	`, now, now-60)))
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
	if n := len(pub.events["client"]); n != 1 {
		t.Fatalf("Expected 1 event to be published. Got %d", n)
	}
	b, err := json.Marshal(pub.events["client"][0])
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
}

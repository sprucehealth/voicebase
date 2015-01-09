package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
)

type stubKeyedRateLimiter struct{}

func (stubKeyedRateLimiter) Check(key string, cost int) (bool, error) {
	if key == "limited" {
		return false, nil
	}
	return true, nil
}

func TestRemoteAddrHandler(t *testing.T) {
	h := RemoteAddrHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), stubKeyedRateLimiter{}, "", metrics.NewRegistry())

	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	req.RemoteAddr = "not-limited"
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected %d, got %d", http.StatusOK, rec.Code)
	}

	rec = httptest.NewRecorder()
	req.RemoteAddr = "limited"
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

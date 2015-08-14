package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type stubKeyedRateLimiter struct{}

func (stubKeyedRateLimiter) Check(key string, cost int) (bool, error) {
	if key == "limited" {
		return false, nil
	}
	return true, nil
}

func TestRemoteAddrHandler(t *testing.T) {
	h := RemoteAddrHandler(httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), stubKeyedRateLimiter{}, "", metrics.NewRegistry())

	req, _ := http.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	req.RemoteAddr = "not-limited"
	h.ServeHTTP(context.Background(), rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected %d, got %d", http.StatusOK, rec.Code)
	}

	rec = httptest.NewRecorder()
	req.RemoteAddr = "limited"
	h.ServeHTTP(context.Background(), rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

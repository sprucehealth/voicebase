package httputil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samuel/go-metrics/metrics"
	"golang.org/x/net/context"
)

func TestMetricsHandler(t *testing.T) {
	reg := metrics.NewRegistry()
	h := MetricsHandler(ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}), reg)
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected %d got %d", http.StatusForbidden, w.Code)
	}
	err = reg.Do(func(name string, value interface{}) error {
		t.Logf("%s %+v", name, value)
		switch name {
		case "requests/response/403":
			if v := value.(*metrics.Counter).Count(); v != 1 {
				return fmt.Errorf("403 response should be 1 got %d", v)
			}
		case "requests/latency":
			if v := value.(metrics.Histogram).Distribution().Count; v != 1 {
				return fmt.Errorf("history count should be 1 got %d", v)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

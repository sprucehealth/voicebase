package cfg

import (
	"net/http"
	"testing"
)

func TestHTTPHandler(t *testing.T) {
	store, err := NewLocalStore([]*ValueDef{
		{
			Name:    "int",
			Type:    ValueTypeInt,
			Default: 123,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var snap Snapshot
	h := HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap = Context(r)
	}), store)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should return default

	h.ServeHTTP(nil, req)
	if v := snap.Int("int"); v != 123 {
		t.Fatalf("Expected 123, got %d", v)
	}

	// Should return updated value

	if err := store.Update(map[string]interface{}{
		"int": 777,
	}); err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(nil, req)
	if v := snap.Int("int"); v != 777 {
		t.Fatalf("Expected 777, got %d", v)
	}
}

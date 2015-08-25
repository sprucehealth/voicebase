package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	JSONResponse(w, http.StatusTeapot, struct{ I int }{I: 123})
	if w.Code != http.StatusTeapot {
		t.Fatalf("Expected %d got %d", http.StatusTeapot, w.Code)
	}
	if e := "{\"I\":123}\n"; w.Body.String() != e {
		t.Fatalf("Expected '%s' got '%s'", e, w.Body.String())
	}
}

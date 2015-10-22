package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type viewCounter struct {
	incrementViewCountErr error
}

func (v *viewCounter) IncrementViewCount(id string) error {
	return v.incrementViewCountErr
}

func TestViewCount(t *testing.T) {
	svc := &viewCounter{}
	h := NewViewCount(svc)

	// No search results
	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", "/", nil)
	test.OK(t, err)
	ctx := mux.SetVars(context.Background(), map[string]string{"id": "foo"})
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
}

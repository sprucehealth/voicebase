package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

func TestFromToContextHandler(t *testing.T) {
	h := ToContextHandler(FromContextHandler(ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {})))
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
}

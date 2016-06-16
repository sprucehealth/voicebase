package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/products"
	"golang.org/x/net/context"
)

func TestProducts(t *testing.T) {
	svc := &productsService{
		lookup: map[string]*products.Product{
			"111": {ID: "111", Name: "blue", ImageURLs: []string{"abc"}, ProductURL: "xxx"},
		},
	}
	h := NewProducts(svc, "", nil)

	// No search results
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	ctx := mux.SetVars(context.Background(), map[string]string{"id": "foo"})
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusNotFound, w)

	// With search results
	w = httptest.NewRecorder()
	r, err = http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	ctx = mux.SetVars(context.Background(), map[string]string{"id": "111"})
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, "{\"product\":{\"id\":\"111\",\"name\":\"blue\",\"image_urls\":[\"abc\"],\"product_url\":\"xxx\"}}\n", w.Body.String())
}

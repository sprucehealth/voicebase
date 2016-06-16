package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/products"
	"golang.org/x/net/context"
)

type productsService struct {
	search map[string][]*products.Product
	lookup map[string]*products.Product
	scrape map[string]*products.Product
}

func (ps *productsService) Search(query string) ([]*products.Product, error) {
	return ps.search[query], nil
}

func (ps *productsService) Lookup(id string) (*products.Product, error) {
	p, ok := ps.lookup[id]
	if !ok {
		return nil, products.ErrNotFound
	}
	return p, nil
}

func (ps *productsService) Scrape(url string) (*products.Product, error) {
	p, ok := ps.scrape[url]
	if !ok {
		return nil, products.ErrNotFound
	}
	return p, nil
}

func TestProductsList(t *testing.T) {
	svc := &productsService{
		search: map[string][]*products.Product{
			"bar": {
				{ID: "111", Name: "blue", ImageURLs: []string{"abc"}, ProductURL: "xxx"},
			},
		},
	}
	h := NewProductsList(svc, "", nil)

	// No search results
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/?q=foo", nil)
	test.OK(t, err)
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, "{\"products\":[]}\n", w.Body.String())

	// With search results
	w = httptest.NewRecorder()
	r, err = http.NewRequest("GET", "/?q=bar", nil)
	test.OK(t, err)
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, "{\"products\":[{\"id\":\"111\",\"name\":\"blue\",\"image_urls\":[\"abc\"],\"product_url\":\"xxx\",\"prefetched\":true}]}\n", w.Body.String())
}

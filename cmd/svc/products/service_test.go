package products

import (
	"testing"

	"github.com/sprucehealth/backend/svc/products"
	"github.com/sprucehealth/backend/test"
)

type testDAL struct {
	queries  map[string][]*products.Product
	products map[string]*products.Product
}

func (td *testDAL) QueryProducts(query string, limit int) ([]*products.Product, error) {
	return td.queries[query], nil
}

func (td *testDAL) Product(id string) (*products.Product, error) {
	p, ok := td.products[id]
	if !ok {
		return nil, products.ErrNotFound
	}
	return p, nil
}

func TestService(t *testing.T) {
	td := &testDAL{
		queries: map[string][]*products.Product{
			"foo": {
				{ID: "111", Name: "some name"},
			},
		},
		products: map[string]*products.Product{
			"111": {ID: "111", Name: "namez"},
		},
	}
	svc := &Service{
		dals:       map[string]products.DAL{"test": td},
		searchDALs: []string{"test"},
	}

	ps, err := svc.Search("")
	test.OK(t, err)
	test.Equals(t, 0, len(ps))
	ps, err = svc.Search("foo")
	test.OK(t, err)
	test.Equals(t, 1, len(ps))
	test.Equals(t, "test:111", ps[0].ID)

	// Non-namespaced ID
	_, err = svc.Lookup("111")
	test.Equals(t, products.ErrNotFound, err)
	// Non-existant data source
	_, err = svc.Lookup("blah:111")
	test.Equals(t, products.ErrNotFound, err)
	// Non-existant object
	_, err = svc.Lookup("test:222")
	test.Equals(t, products.ErrNotFound, err)
	// Success
	prod, err := svc.Lookup("test:111")
	test.OK(t, err)
	test.Equals(t, "test:111", prod.ID)
}

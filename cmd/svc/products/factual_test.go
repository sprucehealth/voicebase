package products

import (
	"sort"
	"testing"

	"github.com/rainycape/memcache"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/factual"
	"github.com/sprucehealth/backend/test"
)

func init() {
	conc.Testing = true
}

type testFactualClient struct {
	queries   map[string][]*factual.Product
	products  map[string]*factual.Product
	crosswalk []*factual.ProductCrosswalk
}

func (fc *testFactualClient) QueryProducts(query string, filters map[string]*factual.Filter, limit int) ([]*factual.Product, error) {
	return fc.queries[query], nil
}

func (fc *testFactualClient) Product(id string) (*factual.Product, error) {
	p, ok := fc.products[id]
	if !ok {
		return nil, factual.ErrNotFound
	}
	return p, nil
}

func (fc *testFactualClient) QueryProductsCrosswalk(filters map[string]*factual.Filter) ([]*factual.ProductCrosswalk, error) {
	return fc.crosswalk, nil
}

type testMemcachedClient struct {
	cache   map[string]*memcache.Item
	getHit  int
	getMiss int
	set     int
}

func (mc *testMemcachedClient) Get(key string) (*memcache.Item, error) {
	it, ok := mc.cache[key]
	if ok {
		mc.getHit++
		return it, nil
	}
	mc.getMiss++
	return nil, memcache.ErrCacheMiss
}

func (mc *testMemcachedClient) Set(item *memcache.Item) error {
	mc.set++
	if mc.cache == nil {
		mc.cache = make(map[string]*memcache.Item)
	}
	mc.cache[item.Key] = item
	return nil
}

func TestSortCrosswalk(t *testing.T) {
	pcw := []*factual.ProductCrosswalk{
		{Namespace: "abc"},
		{Namespace: "target"},
		{Namespace: "amazon"},
		{Namespace: "wallmart"},
	}
	sort.Sort(crosswalkByNamespace(pcw))
	test.Equals(t, "abc", pcw[0].Namespace)
	test.Equals(t, "wallmart", pcw[1].Namespace)
	test.Equals(t, "target", pcw[2].Namespace)
	test.Equals(t, "amazon", pcw[3].Namespace)
}

func TestFactualDAL(t *testing.T) {
	fc := &testFactualClient{
		queries: map[string][]*factual.Product{
			"foo": {
				{
					FactualID:   "111",
					ProductName: "name",
				},
			},
		},
		products: map[string]*factual.Product{
			"111": {
				FactualID:   "111",
				ProductName: "name",
			},
		},
		crosswalk: []*factual.ProductCrosswalk{
			{
				FactualID: "111",
				URL:       "https://example.com",
			},
		},
	}
	mc := &testMemcachedClient{}
	dal := newFactualDAL(fc, mc, metrics.NewRegistry())

	// Non-existant product
	_, err := dal.Product("zzz")
	test.Equals(t, errNotFound, err)
	test.Equals(t, 1, mc.getMiss)
	// Existant product (uncached)
	p, err := dal.Product("111")
	test.OK(t, err)
	test.Equals(t, p.ID, "111")
	test.Equals(t, p.Name, "name")
	test.Equals(t, p.ProductURL, "https://example.com")
	test.Equals(t, 2, mc.getMiss)
	test.Equals(t, 1, mc.set)
	// Existant product (cached)
	p, err = dal.Product("111")
	test.OK(t, err)
	test.Equals(t, p.ID, "111")
	test.Equals(t, p.Name, "name")
	test.Equals(t, p.ProductURL, "https://example.com")
	test.Equals(t, 1, mc.getHit)

	// No products matched
	prods, err := dal.QueryProducts("zzz", 0)
	test.OK(t, err)
	test.Equals(t, 0, len(prods))
	test.Equals(t, 2, mc.set)
	test.Equals(t, 3, mc.getMiss)
	// Products matched (uncached)
	prods, err = dal.QueryProducts("foo", 0)
	test.OK(t, err)
	test.Equals(t, 1, len(prods))
	test.Equals(t, prods[0].ID, "111")
	test.Equals(t, 3, mc.set)
	test.Equals(t, 4, mc.getMiss)
	// Products matched (cached)
	prods, err = dal.QueryProducts("foo", 0)
	test.OK(t, err)
	test.Equals(t, 1, len(prods))
	test.Equals(t, prods[0].ID, "111")
	test.Equals(t, 3, mc.set)
	test.Equals(t, 2, mc.getHit)
}

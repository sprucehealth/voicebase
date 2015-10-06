package products

import (
	"strings"

	"github.com/rainycape/memcache"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/factual"
	"github.com/sprucehealth/backend/svc/products"
)

const queryLimit = 20

// Service is the products service
type Service struct {
	// dals is the set of data access layers for products
	dals map[string]dal
	// searchDAL is the DAL that will be used as the search source
	searchDAL string
}

// dal is the interface for a products data access layer
type dal interface {
	QueryProducts(query string, limit int) ([]*products.Product, error)
	Product(id string) (*products.Product, error)
}

// FactualClient is the interface implemented by a Factual client
type FactualClient interface {
	QueryProducts(query string, filters map[string]*factual.Filter, limit int) ([]*factual.Product, error)
	Product(id string) (*factual.Product, error)
	QueryProductsCrosswalk(filters map[string]*factual.Filter) ([]*factual.ProductCrosswalk, error)
}

// MemcacheClient is the interface implemented by a Memcached client
type MemcacheClient interface {
	Get(key string) (*memcache.Item, error)
	Set(item *memcache.Item) error
}

// New returns a newly initialized products service
func New(fc FactualClient, mc MemcacheClient, statsRegistry metrics.Registry) *Service {
	return &Service{
		dals:      map[string]dal{"factual": newFactualDAL(fc, mc, statsRegistry.Scope("factual"))},
		searchDAL: "factual",
	}
}

// Search returns a list of products that match a search query
func (s *Service) Search(query string) ([]*products.Product, error) {
	dal := s.dals[s.searchDAL]
	prods, err := dal.QueryProducts(query, queryLimit)
	if err != nil {
		return nil, err
	}
	// Tag the product IDs with the source
	for _, p := range prods {
		p.ID = s.searchDAL + ":" + p.ID
	}
	return prods, nil
}

// Lookup returns a single product by ID or products.ErrNotFound if the product does not exist.
func (s *Service) Lookup(id string) (*products.Product, error) {
	ix := strings.IndexByte(id, ':')
	if ix < 0 {
		return nil, products.ErrNotFound
	}
	dalName := id[:ix]
	dal := s.dals[dalName]
	if dal == nil {
		return nil, products.ErrNotFound
	}
	id = id[ix+1:]
	prod, err := dal.Product(id)
	if err == errNotFound {
		return nil, products.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	prod.ID = dalName + ":" + prod.ID
	return prod, nil
}

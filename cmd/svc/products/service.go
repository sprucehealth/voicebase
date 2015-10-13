package products

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rainycape/memcache"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/factual"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/products"
)

const (
	queryLimit             = 20
	scrapeCacheDurationSec = 60 * 60 * 24 * 21
)

// Service is the products service
type Service struct {
	// dals is the set of data access layers for products
	dals map[string]dal
	// searchDAL is the DAL that will be used as the search source
	searchDAL string
	mc        MemcacheClient
	az        AmazonProductClient
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

// AmazonProductClient is a client of the Amazon advertising API.
type AmazonProductClient interface {
	LookupByASIN(asin string) (*products.Product, error)
}

// New returns a newly initialized products service
func New(fc FactualClient, az AmazonProductClient, mc MemcacheClient, statsRegistry metrics.Registry) *Service {
	return &Service{
		az:        az,
		mc:        mc,
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

// Scrape parses the page for the provided URL and returns any product found if any. It returns
// products.ErrScrapeFailed if it is unable to extract a product.
func (s *Service) Scrape(earl string) (*products.Product, error) {
	cacheKey := "scrape:" + earl
	if s.mc != nil {
		if it, err := s.mc.Get(cacheKey); err == nil {
			var p products.Product
			if err := json.Unmarshal(it.Value, &p); err == nil {
				return &p, nil
			}
		}
	}

	u, err := url.Parse(earl)
	if err != nil {
		return nil, products.ErrScrapeFailed{Reason: "failed to parse URL"}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, products.ErrScrapeFailed{Reason: "invalid URL scheme"}
	}
	if r, ok := validate.RemoteHost(u.Host); !ok {
		// Log this so that we can track what URLs are failing
		golog.Warningf("Invalid remote host when scraping '%s': %s", earl, r)
		return nil, products.ErrScrapeFailed{Reason: "invalid host"}
	}

	var prod *products.Product
	if u.Host == "www.amazon.com" {
		if s.az == nil {
			return nil, errors.Trace(fmt.Errorf("products: amazon products client not available"))
		}
		// Get ASIN from URL
		var asin string
		parts := strings.Split(u.Path, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			p := parts[i]
			if len(p) == 10 && !strings.HasPrefix(p, "ref=") {
				asin = p
				break
			}
		}
		prod, err = s.az.LookupByASIN(asin)
		if err != nil {
			return nil, err
		}
	} else {
		res, err := http.Get(earl)
		if err != nil {
			return nil, products.ErrScrapeFailed{Reason: "request failed"}
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil, products.ErrScrapeFailed{Reason: "bad response"}
		}
		if strings.HasPrefix(res.Header.Get("Content-Type"), "image/") {
			return &products.Product{ImageURLs: []string{u.String()}}, nil
		} else if !strings.HasPrefix(res.Header.Get("Content-Type"), "text/") {
			return nil, products.ErrScrapeFailed{Reason: "invalid content type " + res.Header.Get("Content-Type")}
		}
		prod, err = scrape(u, res.Body)
		if err != nil {
			return nil, err
		}
	}

	// Cache result if using memcached
	if s.mc != nil {
		conc.Go(func() {
			if b, err := json.Marshal(prod); err == nil {
				if err := s.mc.Set(&memcache.Item{
					Key:        cacheKey,
					Value:      b,
					Expiration: scrapeCacheDurationSec,
				}); err != nil {
					golog.Errorf("Failed to cache product: %s", err)
				}
			}
		})
	}
	return prod, nil
}

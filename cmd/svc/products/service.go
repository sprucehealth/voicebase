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
	dals map[string]products.DAL
	// searchDALs are the DALs that will be used as the search sources
	searchDALs []string
	mc         MemcacheClient
	az         AmazonProductClient
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
func New(fc FactualClient, az AmazonProductClient, mc MemcacheClient, statsRegistry metrics.Registry, additionalDals map[string]products.DAL) *Service {
	dals := map[string]products.DAL{"factual": newFactualDAL(fc, mc, statsRegistry.Scope("factual"))}
	searchDALs := make([]string, 0, len(additionalDals)+1)
	if additionalDals != nil {
		for k, v := range additionalDals {
			dals[k] = v
			searchDALs = append(searchDALs, k)
		}
	}
	// Search factual last
	// TODO: Make this order configurable
	searchDALs = append(searchDALs, "factual")
	return &Service{
		az:         az,
		mc:         mc,
		dals:       dals,
		searchDALs: searchDALs,
	}
}

// Search returns a list of products that match a search query
func (s *Service) Search(query string) ([]*products.Product, error) {
	var prods []*products.Product
	for _, searchDAL := range s.searchDALs {
		dal := s.dals[searchDAL]
		ps, err := dal.QueryProducts(query, queryLimit)
		if err != nil {
			return nil, err
		}
		// Tag the product IDs with the source
		for _, p := range ps {
			p.ID = searchDAL + ":" + p.ID
		}
		prods = append(prods, ps...)
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
	if err == errNotFound || err == products.ErrNotFound {
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
		return nil, errors.Trace(products.ErrScrapeFailed{Reason: "failed to parse URL"})
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.Trace(products.ErrScrapeFailed{Reason: "invalid URL scheme"})
	}
	if r, ok := validate.RemoteHost(u.Host, true); !ok {
		// Log this so that we can track what URLs are failing
		golog.Warningf("Invalid remote host when scraping '%s': %s", earl, r)
		return nil, errors.Trace(products.ErrScrapeFailed{Reason: "invalid host"})
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
			return nil, errors.Trace(err)
		}
	} else {
		prod, err = fetchAndScrape(u, 0)
		if err != nil {
			return nil, errors.Trace(err)
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

func fetchAndScrape(u *url.URL, depth int) (*products.Product, error) {
	earl := u.String()
	res, err := http.Get(earl)
	if err != nil {
		return nil, errors.Trace(products.ErrScrapeFailed{Reason: "request failed"})
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.Trace(products.ErrScrapeFailed{Reason: fmt.Sprintf("bad response (%d response code)", res.StatusCode)})
	}
	if strings.HasPrefix(res.Header.Get("Content-Type"), "image/") {
		return &products.Product{ImageURLs: []string{earl}}, nil
	} else if !strings.HasPrefix(res.Header.Get("Content-Type"), "text/") {
		return nil, errors.Trace(products.ErrScrapeFailed{Reason: "invalid content type " + res.Header.Get("Content-Type")})
	}
	p, err := scrape(u, res.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// If no images found then try scraping canonical URL. For instance mobile
	// Sephora would require custom rules to scrape, but the canonical URL
	// points to the desktop version which works by default. Only do this for
	// one level of depth though (avoid recursing forever).
	if len(p.ImageURLs) == 0 && depth < 1 && u.String() != p.ProductURL {
		u2, err := url.Parse(p.ProductURL)
		if err == nil {
			p2, err := fetchAndScrape(u2, depth+1)
			if err == nil && len(p2.ImageURLs) != 0 {
				p = p2
			}
		}
	}

	return p, nil
}

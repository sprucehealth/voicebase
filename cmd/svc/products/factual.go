package products

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/rainycape/memcache"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/factual"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/products"
)

const (
	productQueryCacheDurationSec = 14 * 24 * 60 * 60
	productCacheDurationSec      = 14 * 24 * 60 * 60
)

var errNotFound = errors.New("products: object not found")

type factualDAL struct {
	fc                FactualClient
	mc                MemcacheClient
	statMcQueryHit    *metrics.Counter
	statMcQueryMiss   *metrics.Counter
	statMcProductHit  *metrics.Counter
	statMcProductMiss *metrics.Counter
	statMcGetSuccess  *metrics.Counter
	statMcGetFailure  *metrics.Counter
	statMcSetSuccess  *metrics.Counter
	statMcSetFailure  *metrics.Counter
}

func newFactualDAL(fc FactualClient, mc MemcacheClient, statsRegistry metrics.Registry) *factualDAL {
	dal := &factualDAL{
		fc:                fc,
		mc:                mc,
		statMcQueryHit:    metrics.NewCounter(),
		statMcQueryMiss:   metrics.NewCounter(),
		statMcProductHit:  metrics.NewCounter(),
		statMcProductMiss: metrics.NewCounter(),
		statMcGetSuccess:  metrics.NewCounter(),
		statMcGetFailure:  metrics.NewCounter(),
		statMcSetSuccess:  metrics.NewCounter(),
		statMcSetFailure:  metrics.NewCounter(),
	}
	statsRegistry.Add("cache/query/hit", dal.statMcQueryHit)
	statsRegistry.Add("cache/query/miss", dal.statMcQueryMiss)
	statsRegistry.Add("cache/product/hit", dal.statMcProductHit)
	statsRegistry.Add("cache/product/miss", dal.statMcProductMiss)
	statsRegistry.Add("cache/get/success", dal.statMcGetSuccess)
	statsRegistry.Add("cache/get/failure", dal.statMcGetFailure)
	statsRegistry.Add("cache/set/success", dal.statMcSetSuccess)
	statsRegistry.Add("cache/set/failure", dal.statMcSetFailure)
	return dal
}

func (f *factualDAL) QueryProducts(query string, limit int) ([]*products.Product, error) {
	// TODO: could handle limit better by latching to certain values for better cache hit rate
	cacheKey := "facdal:q:" + query + ":" + strconv.Itoa(limit)

	if f.mc != nil {
		if it, err := f.mc.Get(cacheKey); err == nil {
			f.statMcQueryHit.Inc(1)
			f.statMcGetSuccess.Inc(1)
			var prods []*products.Product
			if err := json.Unmarshal(it.Value, &prods); err != nil {
				golog.Errorf("Failed to unmarshal cached products for key '%s': %s", cacheKey, err)
			} else {
				return prods, nil
			}
		} else if err == memcache.ErrCacheMiss {
			f.statMcQueryMiss.Inc(1)
		} else {
			f.statMcGetFailure.Inc(1)
			golog.Errorf("Memcached GET failed: %s", err)
		}
	}

	ps, err := f.fc.QueryProducts(query, nil, limit)
	if err != nil {
		return nil, errors.Trace(err)
	}
	prods := make([]*products.Product, len(ps))
	ids := make([]string, len(ps))
	for i, p := range ps {
		ids[i] = p.FactualID
		prods[i] = &products.Product{
			ID:        p.FactualID,
			Name:      p.ProductName,
			ImageURLs: p.ImageURLs,
		}
	}

	// Lookup product URLs in the crosswalk database
	if len(ids) != 0 {
		urls, err := f.productURLs(ids)
		if err != nil {
			golog.Errorf("Failed to lookup factual products crosswalk: %s", err)
			return prods, nil
		}
		for _, p := range prods {
			p.ProductURL = urls[p.ID]
		}
	}

	// Cache result if using memcached
	if f.mc != nil {
		conc.Go(func() {
			if b, err := json.Marshal(prods); err != nil {
				golog.Errorf("Failed to marshal products: %s", err)
			} else {
				if err := f.mc.Set(&memcache.Item{
					Key:        cacheKey,
					Value:      b,
					Expiration: productQueryCacheDurationSec,
				}); err != nil {
					f.statMcSetFailure.Inc(1)
					golog.Errorf("Failed to cache products: %s", err)
				} else {
					f.statMcSetSuccess.Inc(1)
				}
			}
		})
	}
	return prods, nil
}

func (f *factualDAL) Product(id string) (*products.Product, error) {
	cacheKey := "facdal:p:" + id

	if f.mc != nil {
		if it, err := f.mc.Get(cacheKey); err == nil {
			f.statMcProductHit.Inc(1)
			f.statMcGetSuccess.Inc(1)
			var prod *products.Product
			if err := json.Unmarshal(it.Value, &prod); err != nil {
				golog.Errorf("Failed to unmarshal cached product for key '%s': %s", cacheKey, err)
			} else {
				return prod, nil
			}
		} else if err == memcache.ErrCacheMiss {
			f.statMcProductMiss.Inc(1)
		} else {
			f.statMcGetFailure.Inc(1)
			golog.Errorf("Memcached GET failed: %s", err)
		}
	}

	p, err := f.fc.Product(id)
	if err == factual.ErrNotFound {
		return nil, errNotFound
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	prod := &products.Product{
		ID:        p.FactualID,
		Name:      p.ProductName,
		ImageURLs: p.ImageURLs,
	}

	urls, err := f.productURLs([]string{id})
	if err != nil {
		golog.Errorf("Failed to lookup factual products crosswalk: %s", err)
		return prod, nil
	}
	prod.ProductURL = urls[id]

	// Cache result if using memcached
	if f.mc != nil {
		conc.Go(func() {
			if b, err := json.Marshal(prod); err != nil {
				golog.Errorf("Failed to marshal product: %s", err)
			} else {
				if err := f.mc.Set(&memcache.Item{
					Key:        cacheKey,
					Value:      b,
					Expiration: productCacheDurationSec,
				}); err != nil {
					f.statMcSetFailure.Inc(1)
					golog.Errorf("Failed to cache product: %s", err)
				} else {
					f.statMcSetSuccess.Inc(1)
				}
			}
		})
	}
	return prod, nil
}

func (f *factualDAL) productURLs(ids []string) (map[string]string, error) {
	pcw, err := f.fc.QueryProductsCrosswalk(map[string]*factual.Filter{
		"factual_id": {In: ids},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	sort.Sort(crosswalkByNamespace(pcw))
	urls := make(map[string]string, len(pcw))
	for _, p := range pcw {
		urls[p.FactualID] = p.URL
	}
	return urls, nil
}

type crosswalkByNamespace []*factual.ProductCrosswalk

func (c crosswalkByNamespace) Len() int      { return len(c) }
func (c crosswalkByNamespace) Swap(a, b int) { c[a], c[b] = c[b], c[a] }
func (c crosswalkByNamespace) Less(a, b int) bool {
	return crosswalkNamespacePrecedence[c[a].Namespace] < crosswalkNamespacePrecedence[c[b].Namespace]
}

// crosswalkNamespacePrecedence is the order for crosswalk namespaces, higher precedence gets priority
var crosswalkNamespacePrecedence = map[string]int{
	"amazon":   10,
	"target":   9,
	"wallmart": 8,
}

package mediaproxy

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/rainycape/memcache"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	cacheDefaultDuration = time.Second * 60 * 60 * 24 * 7
	cachePrefix          = "mediaproxy:"
)

// CacheDAL provides a caching layer for a DAL
type CacheDAL struct {
	mc             MemcacheClient
	dal            DAL
	cacheDuration  time.Duration
	statHit        *metrics.Counter
	statMiss       *metrics.Counter
	statGetSuccess *metrics.Counter
	statSetSuccess *metrics.Counter
	statGetFail    *metrics.Counter
	statSetFail    *metrics.Counter
}

// MemcacheClient is the interface implemented by a Memcached client
type MemcacheClient interface {
	GetMulti(keys []string) (map[string]*memcache.Item, error)
	Add(item *memcache.Item) error
	Set(item *memcache.Item) error
}

// NewCacheDAL returns a DAL that provides a caching layer for another DAL
func NewCacheDAL(mc MemcacheClient, underlyingDAL DAL, cacheDuration time.Duration, metricsRegistry metrics.Registry) *CacheDAL {
	if cacheDuration <= 0 {
		cacheDuration = cacheDefaultDuration
	}
	cd := &CacheDAL{
		mc:             mc,
		dal:            underlyingDAL,
		cacheDuration:  cacheDuration,
		statHit:        metrics.NewCounter(),
		statMiss:       metrics.NewCounter(),
		statGetSuccess: metrics.NewCounter(),
		statSetSuccess: metrics.NewCounter(),
		statGetFail:    metrics.NewCounter(),
		statSetFail:    metrics.NewCounter(),
	}
	metricsRegistry.Add("hits", cd.statHit)
	metricsRegistry.Add("misses", cd.statMiss)
	metricsRegistry.Add("get/successes", cd.statGetSuccess)
	metricsRegistry.Add("get/failures", cd.statGetFail)
	metricsRegistry.Add("set/successes", cd.statSetSuccess)
	metricsRegistry.Add("set/failures", cd.statSetFail)
	return cd
}

// Get tries to find the media in the cache first, fetches from the underlying
// DAL anything necessary, and caches anything that was fetched.
func (d *CacheDAL) Get(ids []string) ([]*Media, error) {
	// Check cache first
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cachePrefix + id
	}
	items, err := d.mc.GetMulti(keys)
	if err == nil {
		d.statGetSuccess.Inc(1)
	} else {
		golog.Errorf("mediaproxy: cache GetMulti failed: %s", err)
		d.statGetFail.Inc(1)
	}
	ms := make([]*Media, 0, len(ids))
	found := make(map[string]struct{}, len(items))
	for _, it := range items {
		m := &Media{}
		if err := json.Unmarshal(it.Value, m); err == nil {
			ms = append(ms, m)
			found[m.ID] = struct{}{}
		}
	}

	d.statHit.Inc(uint64(len(ms)))

	// Shortcut if all media was found in the cache
	if len(ids) == len(ms) {
		return ms, nil
	}

	d.statMiss.Inc(uint64(len(ids) - len(ms)))

	// Filter ID list of media that was found in the cache
	for i := 0; i < len(ids); i++ {
		id := ids[i]
		if _, ok := found[id]; ok {
			ids[i] = ids[len(ids)-1]
			ids = ids[:len(ids)-1]
			i--
		}
	}

	// Fetch uncached media from underlying DAL
	fetched, err := d.dal.Get(ids)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Cache newly fetched media (do this in the foreground to mitigate
	// race conditions on read-modify-write in the same process)
	d.put(ms, false)

	// Merge fetched and previously cached media
	for _, m := range fetched {
		ms = append(ms, m)
	}
	return ms, nil
}

// Put writes through the cache to the underlying DAL
func (d *CacheDAL) Put(ms []*Media) error {
	// Doing this in parallel could lead to a situation where the cache has
	// the media stored by the underlying store fails and thus not have the
	// media. Probably find though as it doesn't really matter in the end
	// if the data disappears since it'll be regenerated if needed.
	p := conc.NewParallel()
	p.Go(func() error {
		d.put(ms, true)
		return nil
	})
	p.Go(func() error {
		return d.dal.Put(ms)
	})
	return p.Wait()
}

func (d *CacheDAL) put(ms []*Media, overwrite bool) {
	var wg sync.WaitGroup
	wg.Add(len(ms))
	for _, m := range ms {
		go func(m *Media) {
			defer wg.Done()
			b, err := json.Marshal(m)
			if err == nil {
				fn := d.mc.Set
				if overwrite {
					fn = d.mc.Add
				}
				if err := fn(&memcache.Item{
					Key:        cachePrefix + m.ID,
					Value:      b,
					Expiration: int32(d.cacheDuration / time.Second),
				}); err == nil {
					d.statSetSuccess.Inc(1)
				} else if err != memcache.ErrNotStored { // ErrNotStored is returned when the item exists which is fine. We don't want to overwrite.
					d.statSetFail.Inc(1)
				}
			}
		}(m)
	}
	wg.Wait()
}

package mediaproxy

import (
	"testing"
	"time"

	"github.com/rainycape/memcache"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/test"
)

type testMemcachedClient struct {
	cache   map[string]*memcache.Item
	getHit  int
	getMiss int
	add     int
	set     int
}

func (mc *testMemcachedClient) GetMulti(keys []string) (map[string]*memcache.Item, error) {
	items := make(map[string]*memcache.Item)
	for _, k := range keys {
		it, ok := mc.cache[k]
		if ok {
			mc.getHit++
			items[k] = it
		} else {
			mc.getMiss++
		}
	}
	return items, nil
}

func (mc *testMemcachedClient) Add(item *memcache.Item) error {
	mc.add++
	if mc.cache == nil {
		mc.cache = make(map[string]*memcache.Item)
	}
	if _, ok := mc.cache[item.Key]; ok {
		return memcache.ErrNotStored
	}
	mc.cache[item.Key] = item
	return nil
}

func (mc *testMemcachedClient) Set(item *memcache.Item) error {
	mc.set++
	if mc.cache == nil {
		mc.cache = make(map[string]*memcache.Item)
	}
	mc.cache[item.Key] = item
	return nil
}

func TestCacheDAL(t *testing.T) {
	mdal := NewMemoryDAL()
	mc := &testMemcachedClient{}
	cdal := NewCacheDAL(mc, mdal, time.Hour, metrics.NewRegistry())

	// Non-existant id (not in cache or underlying dal) should return no error but not return a value either
	ms, err := cdal.Get([]string{"test"})
	test.Equals(t, nil, err)
	test.Equals(t, 0, len(ms))
	test.Equals(t, 1, mc.getMiss)

	// Write through new item
	test.OK(t, cdal.Put([]*Media{{ID: "foo", URL: "bar"}}))
	test.Equals(t, 1, mc.set)
	ms, err = cdal.Get([]string{"foo"})
	test.Equals(t, nil, err)
	test.Equals(t, 1, len(ms))
	test.Equals(t, "foo", ms[0].ID)
	test.Equals(t, "bar", ms[0].URL)
	test.Equals(t, 1, mc.getHit)

	// Item exists in underlying store but not cache
	test.OK(t, mdal.Put([]*Media{{ID: "blue", URL: "green"}}))
	ms, err = cdal.Get([]string{"blue"})
	test.Equals(t, nil, err)
	test.Equals(t, 1, len(ms))
	test.Equals(t, "blue", ms[0].ID)
	test.Equals(t, "green", ms[0].URL)
	test.Equals(t, 1, mc.add)
	test.Equals(t, 2, mc.getMiss)

	// Update value by write through
	test.OK(t, cdal.Put([]*Media{{ID: "blue", URL: "green"}}))
	test.Equals(t, 2, mc.set)
	ms, err = cdal.Get([]string{"blue"})
	test.Equals(t, nil, err)
	test.Equals(t, 1, len(ms))
	test.Equals(t, "blue", ms[0].ID)
	test.Equals(t, "green", ms[0].URL)
	test.Equals(t, 2, mc.getHit)
}

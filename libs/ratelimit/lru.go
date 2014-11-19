package ratelimit

import (
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-cache/cache"
)

type LRUKeyed struct {
	newRL func() RateLimiter
	rl    cache.Cache
}

func NewLRUKeyed(size int, newRL func() RateLimiter) *LRUKeyed {
	return &LRUKeyed{
		newRL: newRL,
		rl:    cache.NewLRUCache(size),
	}
}

func (k *LRUKeyed) Check(key string, cost int) (bool, error) {
	rli, err := k.rl.Get(key)
	if err != nil {
		return false, err
	}
	var rl RateLimiter
	if rli == nil {
		rl = k.newRL()
		k.rl.Set(key, rl)
	} else {
		rl = rli.(RateLimiter)
	}
	return rl.Check(cost)
}

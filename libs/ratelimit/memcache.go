package ratelimit

import (
	"strconv"
	"time"

	"gopkgs.com/memcache.v2"
)

// Memcache implements a naïve single interval rate-limiter using memcached. It
// uses a counter for the current interval. This implementation is not ideal
// since it allows bursting within the interval. A better implementation could
// do a sliding window using multi-get across multiple intervals.
type Memcache struct {
	cli *memcache.Client
	max int
	sec int
}

func NewMemcache(cli *memcache.Client, max, sec int) *Memcache {
	return &Memcache{
		cli: cli,
		max: max,
		sec: sec,
	}
}

func (mc *Memcache) Check(prefix string, cost int) (bool, error) {
	if cost > mc.max {
		return false, nil
	}

	iv := time.Now().Unix() / int64(mc.sec)
	key := prefix + ":" + strconv.FormatInt(iv, 16)

	var count uint64
	for {
		if v, err := mc.cli.Increment(key, uint64(cost)); err == nil {
			count = v
			break
		} else if err == memcache.ErrCacheMiss {
			// Item does not exist so add the cost. If the add fails with ErrNotStored
			// then someone beat us to it so we need to try the increment again.
			if err := mc.cli.Add(&memcache.Item{
				Key:        key,
				Value:      []byte(strconv.Itoa(cost)),
				Expiration: int32(mc.sec),
			}); err == nil {
				count = uint64(cost)
				break
			} else if err != memcache.ErrNotStored {
				return false, err
			}
		} else {
			return false, err
		}
	}
	return count <= uint64(mc.max), nil
}

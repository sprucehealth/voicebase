package ratelimit

type RateLimiter interface {
	Check(cost int) (bool, error)
}

type KeyedRateLimiter interface {
	Check(key string, cost int) (bool, error)
}

type RateLimiterFunc func(cost int) (bool, error)

func (rlf RateLimiterFunc) Check(cost int) (bool, error) {
	return rlf(cost)
}

type KeyedRateLimiters map[string]KeyedRateLimiter

func (rl KeyedRateLimiters) Get(name string) KeyedRateLimiter {
	if r := rl[name]; r != nil {
		return r
	}
	return NullKeyed{}
}

func (rl KeyedRateLimiters) GetFixedKey(name, key string) RateLimiter {
	r := rl.Get(name)
	return RateLimiterFunc(func(cost int) (bool, error) {
		return r.Check(key, cost)
	})
}

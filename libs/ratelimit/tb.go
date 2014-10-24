package ratelimit

import (
	"sync"
	"time"
)

type TokenBucket struct {
	rate      int
	size      int
	tokens    int
	lastCheck time.Time
	mu        sync.Mutex
}

func NewTokenBucket(rate, size int) *TokenBucket {
	return &TokenBucket{
		rate:      rate,
		size:      size,
		tokens:    size,
		lastCheck: time.Now(),
	}
}

func (tb *TokenBucket) Check(cost int) (bool, error) {
	if cost > tb.size {
		return false, nil
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	tb.tokens += int(now.Sub(tb.lastCheck).Seconds() * float64(tb.rate))
	tb.lastCheck = now
	if tb.tokens > tb.size {
		tb.tokens = tb.size
	}
	if tb.tokens >= cost {
		tb.tokens -= cost
		return true, nil
	}
	return false, nil
}

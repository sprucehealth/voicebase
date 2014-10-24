package ratelimit

import (
	"sync"
	"time"
)

type Simple struct {
	max      int
	duration time.Duration
	last     time.Time
	count    int
	mu       sync.Mutex
}

func NewSimple(max int, d time.Duration) *Simple {
	return &Simple{
		max:      max,
		last:     time.Now(),
		duration: d,
	}
}

func (s *Simple) Check(cost int) (bool, error) {
	if cost > s.max {
		return false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if d := now.Sub(s.last); d > s.duration {
		s.last = now
		s.count = 0
	}
	if s.count+cost <= s.max {
		s.count += cost
		return true, nil
	}
	return false, nil
}

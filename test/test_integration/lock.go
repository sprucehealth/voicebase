package test_integration

import "sync"

type TestLock struct {
	mu       sync.Mutex
	isLocked bool
}

func (t *TestLock) Wait() bool {
	t.mu.Lock()
	t.isLocked = true
	return true
}

func (t *TestLock) Locked() bool {
	return t.isLocked
}

func (t *TestLock) Release() {
	t.mu.Unlock()
}

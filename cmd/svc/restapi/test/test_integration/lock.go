package test_integration

import "sync"

type TestLock struct {
	internalmu sync.Mutex
	mu         sync.Mutex
	isLocked   bool
}

func (t *TestLock) Wait() bool {
	t.internalmu.Lock()
	defer t.internalmu.Unlock()
	t.mu.Lock()
	t.isLocked = true
	return true
}

func (t *TestLock) Locked() bool {
	t.internalmu.Lock()
	defer t.internalmu.Unlock()
	return t.isLocked
}

func (t *TestLock) Release() {
	t.internalmu.Lock()
	defer t.internalmu.Unlock()
	t.mu.Unlock()
}

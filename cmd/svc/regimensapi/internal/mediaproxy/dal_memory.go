package mediaproxy

import (
	"sync"
)

// MemoryDAL is an in memory store for media metadata. It can be used for
// testing and local development. DO NOT USE ANYWHERE ELSE!
type MemoryDAL struct {
	mu    sync.RWMutex
	media map[string]*Media // id -> media
}

// NewMemoryDAL returns an initialized MemoryDAL
func NewMemoryDAL() *MemoryDAL {
	return &MemoryDAL{
		media: make(map[string]*Media),
	}
}

func (d *MemoryDAL) Get(id []string) ([]*Media, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ms := make([]*Media, 0, len(id))
	for _, id := range id {
		if m := d.media[id]; m != nil {
			ms = append(ms, m)
		}
	}
	return ms, nil
}

func (d *MemoryDAL) Put(ms []*Media) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.media == nil {
		d.media = make(map[string]*Media)
	}
	for _, m := range ms {
		d.media[m.ID] = m
	}
	return nil
}

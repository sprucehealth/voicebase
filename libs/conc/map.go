package conc

import "sync"

// Map provides an interface for any object to conform to
// that provides key/value storage.
type Map interface {
	// Get returns value for key if it exists, nil otherwise.
	Get(key string) interface{}

	// Set sets the value for the key.
	Set(key string, value interface{})

	// Delete delets the value for the key if present.
	Delete(key string)

	// Snapshot returns a current snapshot of the map
	// to make it easy to iterate over.
	Snapshot() map[string]interface{}
}

type concurrentMap struct {
	cmap map[string]interface{}
	mux  sync.RWMutex
}

// NewMap returns a map for concurrent access.
func NewMap() Map {
	return &concurrentMap{
		cmap: make(map[string]interface{}),
	}
}

func (c *concurrentMap) Get(key string) interface{} {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.cmap[key]
}

func (c *concurrentMap) Set(key string, value interface{}) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.cmap[key] = value
}

func (c *concurrentMap) Delete(key string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	delete(c.cmap, key)
}

func (c *concurrentMap) Snapshot() map[string]interface{} {
	c.mux.RLock()
	defer c.mux.RUnlock()
	snapshot := make(map[string]interface{}, len(c.cmap))
	for k, v := range c.cmap {
		snapshot[k] = v
	}

	return snapshot
}

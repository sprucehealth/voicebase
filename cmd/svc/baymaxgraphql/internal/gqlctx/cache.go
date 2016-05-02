package gqlctx

import (
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
)

// EntityCache represents a thread safe map of key to entity objects we have encountered
// This cache is intended to be used in conjunction with request context
type EntityCache struct {
	cMap conc.Map
}

// NewEntityCache returns an initialized instance of EntityCache
func NewEntityCache(ini map[string]*directory.Entity) *EntityCache {
	cMap := conc.NewMap()
	for k, e := range ini {
		cMap.Set(k, e)
	}
	return &EntityCache{
		cMap: cMap,
	}
}

// Get returns the entity mapped to the provided key and nil if it does not exist
func (c *EntityCache) Get(key string) *directory.Entity {
	ei := c.cMap.Get(key)
	if ei == nil {
		return nil
	}

	ent, ok := ei.(*directory.Entity)
	if !ok {
		golog.Errorf("EntityCache: Found %+v mapped to %s but failed conversion from interface to Entity. Removing from cache.", ei, key)
		go c.cMap.Delete(key)
		return nil
	}
	return ent
}

// Set maps the provided entity to the provided key
func (c *EntityCache) Set(key string, ent *directory.Entity) {
	c.cMap.Set(key, ent)
}

// Delete removed the provided key from the cache
func (c *EntityCache) Delete(key string) {
	c.cMap.Delete(key)
}

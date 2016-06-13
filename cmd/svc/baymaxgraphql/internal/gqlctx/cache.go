package gqlctx

import (
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
)

// EntityGroupCache represents a thread safe map of key to entity objects we have encountered that map to that key
// This cache is intended to be used in conjunction with request context
type EntityGroupCache struct {
	cMap *conc.Map
}

// NewEntityGroupCache returns an initialized instance of NewEntityGroupCache
func NewEntityGroupCache(ini map[string][]*directory.Entity) *EntityGroupCache {
	cMap := conc.NewMap()
	for k, es := range ini {
		cMap.Set(k, es)
	}
	return &EntityGroupCache{
		cMap: cMap,
	}
}

// Get returns the entity mapped to the provided key and nil if it does not exist
func (c *EntityGroupCache) Get(key string) []*directory.Entity {
	esi := c.cMap.Get(key)
	if esi == nil {
		return nil
	}

	ents, ok := esi.([]*directory.Entity)
	if !ok {
		golog.Errorf("EntityGroupCache: Found %+v mapped to %s but failed conversion from interface to []*Entity. Removing from cache.", esi, key)
		go c.cMap.Delete(key)
		return nil
	}
	return ents
}

// GetOnly returns the only entity mapped to the provided key, nil if no entities exist or more than 1 is mapped to the key
// An error is logged in the case that multiple entities are mapped to a key that is provided
func (c *EntityGroupCache) GetOnly(key string) *directory.Entity {
	ents := c.Get(key)
	if len(ents) == 0 {
		return nil
	}
	if len(ents) > 1 {
		golog.Errorf("Expected only 1 entity to be present in EntityGroupCache for key %s, but found %v", key, ents)
		return nil
	}
	return ents[0]
}

// Set maps the provided entity to the provided key
func (c *EntityGroupCache) Set(key string, ent *directory.Entity) {
	c.cMap.Set(key, []*directory.Entity{ent})
}

// SetGroup maps the provided entity slice to the provided key
func (c *EntityGroupCache) SetGroup(key string, ents []*directory.Entity) {
	c.cMap.Set(key, ents)
}

// Delete removes the provided key from the cache
func (c *EntityGroupCache) Delete(key string) {
	c.cMap.Delete(key)
}

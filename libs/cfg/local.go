package cfg

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/sprucehealth/backend/libs/golog"
)

type localStore struct {
	defs     map[string]*ValueDef
	values   atomic.Value
	updateMu sync.Mutex
}

// NewLocalStore returns an instance of a config store that stores values
// in memory. It it safe for concurrent access and can be used for testing
// or as the cache for distributed store implementations.
func NewLocalStore(defs []*ValueDef) (Store, error) {
	lc := &localStore{
		defs: make(map[string]*ValueDef, len(defs)),
	}
	for _, d := range defs {
		if err := lc.Register(d); err != nil {
			return nil, err
		}
	}
	lc.storeValues(make(map[string]interface{}))
	return lc, nil
}

func (lc *localStore) Close() error {
	return nil
}

func (lc *localStore) Register(def *ValueDef) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("config.LocalConfig: %+v is not a valid definition: %s", def, err)
	}
	if _, ok := lc.defs[def.Name]; ok {
		return fmt.Errorf("config.LocalConfig: name %s already registered", def.Name)
	}
	lc.defs[def.Name] = def
	return nil
}

func (lc *localStore) Defs() map[string]*ValueDef {
	return lc.defs
}

func (lc *localStore) Snapshot() Snapshot {
	return Snapshot{
		values: lc.loadValues(),
		defs:   lc.defs,
	}
}

func (lc *localStore) Update(update map[string]interface{}) error {
	return lc.update(update, true)
}

func (lc *localStore) loadValues() map[string]interface{} {
	return lc.values.Load().(map[string]interface{})
}

func (lc *localStore) storeValues(values map[string]interface{}) {
	lc.values.Store(values)
}

func (lc *localStore) update(update map[string]interface{}, logUnknown bool) error {
	lc.updateMu.Lock()
	defer lc.updateMu.Unlock()
	oldValues := lc.loadValues()
	newValues := make(map[string]interface{}, len(oldValues))
	// Clone the map. Since we know that the values are immutable types we
	// can just copy them over.
	for n, v := range oldValues {
		newValues[n] = v
	}
	for name, v := range update {
		def, ok := lc.defs[name]
		if !ok {
			if logUnknown {
				golog.Errorf("config.LocalConfig.Update: no definition registered for '%s'", name)
			}
			newValues[name] = v
			continue
		}
		v, ok = normalizeType(v, def.Type, false)
		if !ok {
			golog.Errorf("config.LocalConfig.Update: wrong type trying to update '%s', wanted %s got %T",
				name, def.Type, v)
			continue
		}
		newValues[name] = v
	}
	lc.storeValues(newValues)
	return nil
}

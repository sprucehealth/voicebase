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

func NewLocalStore() Store {
	lc := &localStore{
		defs: make(map[string]*ValueDef),
	}
	lc.storeValues(make(map[string]interface{}))
	return lc
}

func (lc *localStore) Close() error {
	return nil
}

func (lc *localStore) Register(def *ValueDef) {
	if !def.Valid() {
		panic(fmt.Sprintf("config.LocalConfig: %+v is not a valid definition", def))
	}
	if _, ok := lc.defs[def.Name]; ok {
		panic("config.LocalConfig: name " + def.Name + " already registered")
	}
	lc.defs[def.Name] = def
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
		v, ok = normalizeType(v, def.Type)
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

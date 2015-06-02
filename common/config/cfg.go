package config

import "github.com/sprucehealth/backend/libs/cfg"

// cfgDefs is a global set of cfg value definitions. It allows packages to register
// values during init and for the set to be used when creating a store.
var cfgDefs = make(map[string]*cfg.ValueDef)

// CfgDefs returns the global set of cfg value definitions. It allows packages
// to register values during init and for the set to be used when creating a store.
func CfgDefs() []*cfg.ValueDef {
	defs := make([]*cfg.ValueDef, 0, len(cfgDefs))
	for _, d := range cfgDefs {
		defs = append(defs, d)
	}
	return defs
}

// MustRegisterCfgDef registers a value definition with the global Defs. If the
// provided definition is either invalid or one with the same name is already
// registered then it panics.
func MustRegisterCfgDef(def *cfg.ValueDef) {
	if err := def.Validate(); err != nil {
		panic("config.MustRegisterCfgDef: invalid definition: " + err.Error())
	}
	if _, ok := cfgDefs[def.Name]; ok {
		panic("config.MustRegisterCfgDef: duplicate definition " + def.Name)
	}
	cfgDefs[def.Name] = def
}

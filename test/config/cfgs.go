package config

import "github.com/sprucehealth/backend/libs/cfg"

// GlobalFirstVisitFreeDisabled is a disabled test config
var GlobalFirstVisitFreeDisabled = &cfg.ValueDef{
	Name:        "Global.First.Visit.Free.Enabled",
	Description: "A value that represents if the first visit should be free for all patients.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

// GlobalFirstVisitFreeDisabled is an enabled test config
var GlobalFirstVisitFreeEnabled = &cfg.ValueDef{
	Name:        "Global.First.Visit.Free.Enabled",
	Description: "A value that represents if the first visit should be free for all patients.",
	Type:        cfg.ValueTypeBool,
	Default:     true,
}

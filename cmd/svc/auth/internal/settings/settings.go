package settings

import (
	"github.com/sprucehealth/backend/svc/settings"
)

const (
	ConfigKey2FAEnabled = "2fa_enabled"
)

var Enable2FAConfig = &settings.Config{
	Title:          "Enable 2FA",
	Key:            ConfigKey2FAEnabled,
	AllowSubkeys:   false,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ACCOUNT},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				// TODO: Make on by default in prod and off by default in non-prod.
				Value: false,
			},
		},
	},
}

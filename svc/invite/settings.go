package invite

import "github.com/sprucehealth/backend/svc/settings"

const (
	ConfigKeyOrganizationCode = "organization_code_enabled"
)

// OrganizationCodeConfig represents the config for configuring whether organization codes are available for the org
var OrganizationCodeConfig = &settings.Config{
	Title:          "Enable/Disable organization code availability at org level",
	AllowSubkeys:   false,
	Key:            ConfigKeyOrganizationCode,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: false,
			},
		},
	},
}

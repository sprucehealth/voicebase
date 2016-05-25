package settings

import "github.com/sprucehealth/backend/svc/settings"

const (
	ConfigKeyOptionalTriage = "optional_triage"
)

// OptionalTriageConfig is a settings config to turn all triage points to no-ops.
var OptionalTriageConfig = &settings.Config{
	Title:          "Turn off pre-submission triage",
	Description:    "True indicates that a triage point should result in a no-op and the patient should continue with answering questions in the visit.",
	AllowSubkeys:   false,
	Key:            ConfigKeyOptionalTriage,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Type:           settings.ConfigType_BOOLEAN,
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: false,
			},
		},
	},
}

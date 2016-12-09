package settings

import (
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

// Keys used for registering settings
const (
	ConfigKeyRevealSenderAcrossExcomms = "reveal_sender_across_excomms"
	ConfigKeyProvisionedEndpointTags   = "provisioned_endpoint_tags"
)

// RevealSenderAcrossExCommsConfig represents the config control whether or not we reveal the sender
// when communicating over external services (Email and SMS).
var RevealSenderAcrossExCommsConfig = &settings.Config{
	Title:          "Reveal sender across SMS & Email",
	AllowSubkeys:   false,
	Key:            ConfigKeyRevealSenderAcrossExcomms,
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

// ProvisionedEndpointTagsConfig configures the tags used with a provisioned endpoint when creating threads.
var ProvisionedEndpointTagsConfig = &settings.Config{
	Title:          "Tags used when created a thread that comes through a provisioned endpoint",
	Key:            ConfigKeyProvisionedEndpointTags,
	AllowSubkeys:   true,
	Type:           settings.ConfigType_STRING_LIST,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	OptionalValue:  true,
	Config: &settings.Config_StringList{
		StringList: &settings.StringListConfig{
			Requirements: &settings.StringListRequirements{
				TextRequirements: &settings.TextRequirements{
					MatchRegexp: threading.RegexpValidTag,
				},
			},
		},
	},
}

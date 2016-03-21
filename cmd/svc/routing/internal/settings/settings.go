package settings

import "github.com/sprucehealth/backend/svc/settings"

const (
	ConfigKeyRevealSenderAcrossExcomms = "reveal_sender_across_excomms"
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

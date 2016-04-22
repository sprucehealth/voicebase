package settings

import "github.com/sprucehealth/backend/svc/settings"

const (
	ConfigKeyTeamConversations  = "team_conversations_enabled"
	ConfigKeyCreateSecureThread = "secure_threads_enabled"
)

// TeamConversationsConfig represents the config controlling whether or not team conversations is enabled at the org level
var TeamConversationsConfig = &settings.Config{
	Title:          "Enable/disable team conversations",
	AllowSubkeys:   false,
	Key:            ConfigKeyTeamConversations,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_ORGANIZATION},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

// SecureThreadsConfig represents the config controlling whether or not secure conversations are enabled at the org level
var SecureThreadsConfig = &settings.Config{
	Title:          "Enable/disable secure threads",
	AllowSubkeys:   false,
	Key:            ConfigKeyCreateSecureThread,
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

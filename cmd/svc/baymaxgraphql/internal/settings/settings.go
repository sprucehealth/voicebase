package settings

import (
	"github.com/sprucehealth/backend/environment"

	"github.com/sprucehealth/backend/svc/settings"
)

const (
	ConfigKeyTeamConversations = "team_conversations_enabled"
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
				Value: !environment.IsProd(), // disabled in prod by default, enabled everywhere else.
			},
		},
	},
}

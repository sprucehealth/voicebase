package settings

import (
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

// ClearTextMessageNotificationsConfig represents the config controlling if notifications are disabled or not
var ClearTextMessageNotificationsConfig = &settings.Config{
	Title:          "Show message in notifications",
	AllowSubkeys:   false,
	Key:            threading.ClearTextMessageNotifications,
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

// AlertAllMessagesConfig represents the config controlling if all messages should generate alerts or not
var AlertAllMessagesConfig = &settings.Config{
	Title:          "Alert for all new messages",
	AllowSubkeys:   false,
	Key:            threading.AlertAllMessages,
	Type:           settings.ConfigType_BOOLEAN,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
	Config: &settings.Config_Boolean{
		Boolean: &settings.BooleanConfig{
			Default: &settings.BooleanValue{
				Value: true,
			},
		},
	},
}

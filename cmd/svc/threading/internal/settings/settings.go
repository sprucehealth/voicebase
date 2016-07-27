package settings

import (
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

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

// PreviewPatientMessageContentInNotificationConfig represents the config controlling when the actual content
// for team messages is sent as part of the notification payload.
var PreviewPatientMessageContentInNotificationConfig = &settings.Config{
	Title:          "Show preview for patient messages",
	AllowSubkeys:   false,
	Key:            threading.PreviewPatientMessageContentInNotification,
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

// PreviewTeamMessageContentInNotificationConfig represents the config controlling when the actual content
// for team messages is sent as part of the notification payload.
var PreviewTeamMessageContentInNotificationConfig = &settings.Config{
	Title:          "Show preview for team messages",
	AllowSubkeys:   false,
	Key:            threading.PreviewTeamMessageContentInNotification,
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

package settings

import (
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
)

// ReceiveNotificationsConfig represents the config controlling if notifications are disabled or not
var ReceiveNotificationsConfig = &settings.Config{
	Title:          "Receive Push Notifications",
	AllowSubkeys:   false,
	Key:            notification.ReceiveNotificationsSettingsKey,
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

var threadActivityNotificationPreferenceSingleSelect = &settings.Config_SingleSelect{
	SingleSelect: &settings.SingleSelectConfig{
		Items: []*settings.Item{
			{
				ID:    notification.ThreadActivityNotificationPreferenceAllMessages,
				Label: "All messages",
			},
			{
				ID:    notification.ThreadActivityNotificationPreferenceReferencedOnly,
				Label: "@ only",
			},
			{
				ID:    notification.ThreadActivityNotificationPreferenceOff,
				Label: "Notifications off",
			},
		},
		Default: &settings.SingleSelectValue{
			Item: &settings.ItemValue{
				ID: notification.ThreadActivityNotificationPreferenceAllMessages,
			},
		},
	},
}

// TeamNotificationPreferenceConfig represents the config controlling when notifications are sent for activity on team threads
var TeamNotificationPreferenceConfig = &settings.Config{
	Title:          "Team Conversations",
	AllowSubkeys:   false,
	Key:            notification.TeamNotificationPreferencesSettingsKey,
	Type:           settings.ConfigType_SINGLE_SELECT,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
	Config:         threadActivityNotificationPreferenceSingleSelect,
}

// PatientNotificationPreferenceConfig represents the config controlling when notifications are sent for activity on patient threads
var PatientNotificationPreferenceConfig = &settings.Config{
	Title:          "Patient Conversations",
	AllowSubkeys:   false,
	Key:            notification.PatientNotificationPreferencesSettingsKey,
	Type:           settings.ConfigType_SINGLE_SELECT,
	PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
	Config:         threadActivityNotificationPreferenceSingleSelect,
}

// BadgeCountConfig represents the config controlling whether the badge count is enabled or disabled for
// a provider using Spruce. The badge count is maintained on a per platform basis so that the provider
// can configure whether to enable/disable on iOS independent of android an web.
var BadgeCountConfig = &settings.Config{
	Title:          "Enable/disable badge count for app",
	AllowSubkeys:   true,
	Key:            notification.BadgeCount,
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

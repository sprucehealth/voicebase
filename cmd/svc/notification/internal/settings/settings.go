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

var (
	ThreadActivityNotificationPreferenceAllMessages    = "notification_preference_all"
	ThreadActivityNotificationPreferenceReferencedOnly = "notification_preference_referenced_only"
	ThreadActivityNotificationPreferenceOff            = "notification_preference_off"
)

var threadActivityNotificationPreferenceSingleSelect = &settings.Config_SingleSelect{
	SingleSelect: &settings.SingleSelectConfig{
		Items: []*settings.Item{
			{
				ID:    ThreadActivityNotificationPreferenceAllMessages,
				Label: "All messages",
			},
			{
				ID:    ThreadActivityNotificationPreferenceReferencedOnly,
				Label: "@ only",
			},
			{
				ID:    ThreadActivityNotificationPreferenceOff,
				Label: "Notifications off",
			},
		},
		Default: &settings.SingleSelectValue{
			Item: &settings.ItemValue{
				ID: ThreadActivityNotificationPreferenceAllMessages,
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
